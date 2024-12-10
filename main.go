package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"sync"
	"sync/atomic"
)

type Deck struct {
	CardsLeftByValue [10]int // Cards[0]=A; Cards[1]=2; Cards[2]=3; ...; Cards[9]=10,J,Q,K
	CardsLeft        int
}

func CreateDeck(nDecks int) Deck {
	deck := Deck{}
	for i := 0; i <= 8; i++ {
		deck.CardsLeftByValue[i] = 4 * nDecks
	}
	deck.CardsLeftByValue[9] = 16 * nDecks
	deck.CardsLeft = 52 * nDecks
	return deck
}

type Hand struct {
	Score byte
	Aces  byte
}

func DrawCard(card byte, d *Deck, h *Hand) {
	d.CardsLeftByValue[card]--
	d.CardsLeft--
	if card == AceCard {
		h.Score += 11
		h.Aces++
	} else {
		h.Score += card + 1
	}
	for h.Score > 21 && h.Aces > 0 {
		h.Score -= 10
		h.Aces--
	}
}

func RemoveCard(card byte, d *Deck) {
	d.CardsLeft--
	d.CardsLeftByValue[card]--
}

func ReturnCard(card byte, d *Deck) {
	d.CardsLeft++
	d.CardsLeftByValue[card]++
}

func DrawRandom(d *Deck, h *Hand) byte {
	pos := rand.Intn(d.CardsLeft)
	i := byte(0)
	sum := 0
	for {
		sum += d.CardsLeftByValue[i]
		if sum > pos {
			break
		}
		i++
	}
	DrawCard(i, d, h)
	return i
}

func BustThreshold(h *Hand) byte {
	return 21 - (h.Score - h.Aces*10)
}

func CardMinValue(i byte) byte {
	return i + 1
}

func CardMaxValue(i byte) byte {
	if i == 0 {
		return 11
	}
	return i + 1
}

func MaxProfit(p [3]float64) float64 {
	Max := p[0]
	for i := 1; i < len(p); i++ {
		Max = max(p[i], Max)
	}
	return Max
}

func ResolveDealer(deck *Deck, player, dealer Hand, odds float64) float64 {
	avg := 0.0
	for i := AceCard; i <= TenCard; i++ {
		if deck.CardsLeftByValue[i] == 0 {
			continue
		}
		updatedOdds := odds * (float64(deck.CardsLeftByValue[i]) / float64(deck.CardsLeft))
		d2 := dealer
		DrawCard(i, deck, &d2)

		if d2.Score != 21 {
			var resolveDealer func(Hand, float64) float64
			resolveDealer = func(dealer Hand, odds float64) float64 {
				if dealer.Score < 17 {
					if odds < 1.0/1000000 {
						deckCopy := *deck
						for dealer.Score < 17 {
							DrawRandom(&deckCopy, &dealer)
						}
						res := resolveDealer(dealer, odds)
						return res
					}

					sum := 0.0
					bustThresh := BustThreshold(&dealer)
					for i := AceCard; i <= TenCard; i++ {
						if deck.CardsLeftByValue[i] == 0 {
							continue
						}
						updatedOdds := odds * (float64(deck.CardsLeftByValue[i]) / float64(deck.CardsLeft))
						if CardMinValue(i) > bustThresh {
							sum += winProfit * updatedOdds
						} else {
							d2 := dealer
							DrawCard(i, deck, &d2)
							sum += resolveDealer(d2, updatedOdds)
							ReturnCard(i, deck)
						}
					}
					return sum

				} else if player.Score > dealer.Score {
					return winProfit * odds
				} else if dealer.Score > player.Score {
					return lossProfit * odds
				}
				return tieProfit * odds
			}
			avg += resolveDealer(d2, updatedOdds)
		} else {
			avg += lossProfit * updatedOdds
		}
		ReturnCard(i, deck)
	}

	return avg
}

func Simulate(deck *Deck, cards []byte, dealerCard byte) ([3]float64, byte) {
	player := Hand{}
	for _, c := range cards {
		DrawCard(c, deck, &player)
	}

	dealer := Hand{}
	DrawCard(dealerCard, deck, &dealer)

	var evs [3]float64

	if player.Score == 21 {
		tieChance := 0.0
		if dealer.Score == 10 {
			tieChance = float64(deck.CardsLeftByValue[AceCard]) / float64(deck.CardsLeft)
		} else if dealer.Score == 11 {
			tieChance = float64(deck.CardsLeftByValue[TenCard]) / float64(deck.CardsLeft)
		}
		ev := tieChance*tieProfit + (1.0-tieChance)*blackjackProfit
		for i := 0; i < 3; i++ {
			evs[i] = ev
		}
		return evs, player.Score
	} else {
		var rec func(Hand, *Deck, float64) (float64, float64)
		rec = func(player Hand, deck *Deck, odds float64) (float64, float64) {
			standAvg := ResolveDealer(deck, player, dealer, odds)
			hitAvg := 0.0
			doubleDownAvg := 0.0

			if odds < 1.0/10000 {
				// Prevents deep and pointless recursion chains by forcing standing
				hitAvg = -9999
			} else {
				bustThresh := BustThreshold(&player)
				for i := AceCard; i <= TenCard; i++ {
					if deck.CardsLeftByValue[i] == 0 {
						continue
					}
					updatedOdds := odds * (float64(deck.CardsLeftByValue[i]) / float64(deck.CardsLeft))
					if CardMinValue(i) > bustThresh {
						hitAvg += lossProfit * updatedOdds
						doubleDownAvg += lossProfit * updatedOdds
					} else {
						p2 := player
						DrawCard(i, deck, &p2)
						hitThenStandAvg, hitTwiceAvg := rec(p2, deck, updatedOdds)
						hitAvg += max(hitThenStandAvg, hitTwiceAvg)
						doubleDownAvg += hitThenStandAvg
						ReturnCard(i, deck)
					}
				}

				if odds == 1 {
					evs[standIndex] = standAvg
					evs[hitIndex] = hitAvg
					evs[doubleIndex] = 2 * doubleDownAvg
				}
			}

			return standAvg, hitAvg
		}
		rec(player, deck, 1)
		return evs, player.Score
	}
}

func CalcPerms(nDecks int, card1, card2, dealerCard byte) float64 {
	drawnCards := []byte{card1, card2, dealerCard}
	tens := 0
	repeats := 0
	slices.Sort(drawnCards)
	for i := 0; i < len(drawnCards); i++ {
		if drawnCards[i] == TenCard {
			tens++
		} else if i > 0 && drawnCards[i] == drawnCards[i-1] {
			repeats++
		}
	}
	neither := len(drawnCards) - tens - repeats
	var permutations float64
	if card1 == card2 {
		permutations = 1
	} else {
		permutations = 2
	}
	for i := 0; i < neither; i++ {
		permutations *= float64(4 * nDecks)
	}
	for i := 1; i <= repeats; i++ {
		permutations *= float64(4*nDecks - i)
	}
	for i := 0; i < tens; i++ {
		permutations *= float64(16*nDecks - i)
	}
	return permutations
}

func main() {
	nDecksPtr := flag.Int("nDecks", 1, "Number of decks")
	flag.Parse()

	nDecks := *nDecksPtr
	baseDeck := CreateDeck(nDecks)
	var noAceTable [21 - minScoreWithoutAce][10][3]float64
	var aceTable [21 - minScoreWithAce][10][3]float64
	var splitTable [10][10][2]float64
	var avgPerHand [55][10]float64
	var completed int32

	tenSplitChance := float64(4*nDecks-1) / float64(16*nDecks-1)

	var wg sync.WaitGroup
	wg.Add(550)

	handIndex := 0
	for card1 := AceCard; card1 <= TenCard; card1++ {
		for card2 := AceCard; card2 <= card1; card2++ {
			for dealerCard := AceCard; dealerCard <= TenCard; dealerCard++ {
				card1 := card1
				card2 := card2
				dealerCard := dealerCard
				handIndex := handIndex

				go func() {
					defer wg.Done()
					deck := baseDeck
					profits, handScore := Simulate(&deck, []byte{card1, card2}, dealerCard)
					profit := MaxProfit(profits)

					if handScore != 21 {
						for i := 0; i < 3; i++ {
							if card1 == 0 || card2 == 0 {
								aceTable[handScore-minScoreWithAce][dealerCard][i] += profits[i]
							} else {
								noAceTable[handScore-minScoreWithoutAce][dealerCard][i] += profits[i]
							}
						}
					}

					if card1 == card2 {
						deck := baseDeck
						RemoveCard(card2, &deck)
						splitProfits, _ := Simulate(&deck, []byte{card1}, dealerCard)
						splitProfit := 2 * MaxProfit(splitProfits)

						bestProfit := max(profit, splitProfit)
						splitTable[card1][dealerCard][1] += splitProfit
						splitTable[card1][dealerCard][0] += profit

						if card1 == TenCard {
							profit = profit*(1-tenSplitChance) + bestProfit*tenSplitChance
						} else {
							profit = bestProfit
						}
					}

					avgPerHand[handIndex][dealerCard] = profit

					completed := atomic.AddInt32(&completed, 1)
					fmt.Printf("%3d/550\n", completed)
				}()
			}
			handIndex++
		}
	}
	wg.Wait()

	avgProfit := 0.0
	handIndex = 0
	for card1 := AceCard; card1 <= TenCard; card1++ {
		for card2 := AceCard; card2 <= card1; card2++ {
			for dealerCard := AceCard; dealerCard <= TenCard; dealerCard++ {
				avgProfit += avgPerHand[handIndex][dealerCard] * CalcPerms(nDecks, card1, card2, dealerCard)
			}
			handIndex++
		}
	}

	totalPerms := float64(52*nDecks) * float64(52*nDecks-1) * float64(52*nDecks-2)
	avgProfit /= totalPerms
	fmt.Printf("Average profit: %.5f\n", avgProfit)

	resFile, err := os.Create("results.js")
	if err != nil {
		panic("oh no")
	}

	noAceTableJSON, _ := json.Marshal(noAceTable)
	aceTableJSON, _ := json.Marshal(aceTable)
	splitTableJSON, _ := json.Marshal(splitTable)
	avgPerHandJSON, _ := json.Marshal(avgPerHand)

	jsons := map[string][]byte{
		"noAceTable": noAceTableJSON,
		"aceTable":   aceTableJSON,
		"splitTable": splitTableJSON,
		"avgPerHand": avgPerHandJSON,
	}
	correctOrder := []string{"noAceTable", "aceTable", "splitTable", "avgPerHand"}

	var fileContents []byte
	for _, k := range correctOrder {
		if len(fileContents) > 0 {
			fileContents = append(fileContents, []byte("\n\n")...)
		}
		fileContents = append(fileContents, []byte("const "+k+" = ")...)
		fileContents = append(fileContents, jsons[k]...)
		fileContents = append(fileContents, ';')
	}
	resFile.Write(fileContents)
}
