package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcPerms(t *testing.T) {
	// Cartas repetidas são mais raras
	assert.True(t, CalcPerms(1, SevenCard, SevenCard, SevenCard) < CalcPerms(1, SevenCard, SevenCard, EightCard))
	assert.True(t, CalcPerms(1, SevenCard, SevenCard, EightCard) < CalcPerms(1, SevenCard, EightCard, NineCard))

	// Cartas != 10 tem a mesma chance
	assert.True(t, CalcPerms(1, AceCard, TwoCard, ThreeCard) == CalcPerms(1, FourCard, FiveCard, SixCard))

	// Cartas 10 são mais comuns
	assert.True(t, CalcPerms(1, SevenCard, SevenCard, SevenCard) < CalcPerms(1, TenCard, TenCard, TenCard))
	assert.True(t, CalcPerms(1, AceCard, TwoCard, ThreeCard) < CalcPerms(1, FourCard, FiveCard, TenCard))

	// Total de permutações
	for nDecks := 1; nDecks <= 8; nDecks++ {
		permCount := 0.0
		for card1 := AceCard; card1 <= TenCard; card1++ {
			for card2 := AceCard; card2 <= card1; card2++ {
				for dealerCard := AceCard; dealerCard <= TenCard; dealerCard++ {
					permCount += CalcPerms(nDecks, card1, card2, dealerCard)
				}
			}
		}
		assert.Equal(t, permCount, float64(52*nDecks)*float64(52*nDecks-1)*float64(52*nDecks-2))
	}
}

func TestSimulate(t *testing.T) {
	deck := CreateDeck(1)

	// Hit em mãos de valor baixo
	res, value := Simulate(&deck, []byte{TwoCard, TwoCard}, SixCard)
	assert.Equal(t, value, byte(4))
	assert.True(t, res[hitIndex] > res[standIndex] && res[hitIndex] > res[doubleIndex])

	// Stand em mãos de valor alto
	res, value = Simulate(&deck, []byte{NineCard, NineCard}, SixCard)
	assert.Equal(t, value, byte(18))
	assert.True(t, res[standIndex] > res[hitIndex] && res[standIndex] > res[doubleIndex])

	// Double down em mãos próximas de 11 + dealer com carta baixa
	res, value = Simulate(&deck, []byte{FourCard, SevenCard}, SixCard)
	assert.Equal(t, value, byte(11))
	assert.True(t, res[doubleIndex] > res[standIndex] && res[doubleIndex] > res[hitIndex])

	// Blackjack
	res, value = Simulate(&deck, []byte{AceCard, TenCard}, SixCard)
	assert.Equal(t, value, byte(21))
	assert.Equal(t, res[0], blackjackProfit)
}

func TestResolveDealer(t *testing.T) {
	deck := CreateDeck(1)

	player1 := Hand{
		Score: 2,
		Aces:  0,
	}
	player2 := Hand{
		Score: 16,
		Aces:  0,
	}
	dealer := Hand{
		Score: 4,
		Aces:  0,
	}

	// Pontuações menores que 17 são equivalentes
	assert.True(t, ResolveDealer(&deck, player1, dealer, 1) == ResolveDealer(&deck, player2, dealer, 1))

	player1 = Hand{
		Score: 18,
		Aces:  0,
	}
	player2 = Hand{
		Score: 19,
		Aces:  0,
	}

	// Pontuações maiores são progressivamente melhores
	assert.True(t, ResolveDealer(&deck, player1, dealer, 1) < ResolveDealer(&deck, player2, dealer, 1))
}
