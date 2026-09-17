package domain

import (
	"errors"
	"time"
)

const (
	StartingCoins = 200
	RollCost      = 200
	SellPrice     = 100
	DailyCoins    = 400
)

var (
	ErrInsufficientCoins = errors.New("not enough coins")
	ErrNegativeAmount    = errors.New("amount must be positive")
)

type PlayerKey struct {
	GuildID string
	UserID  string
}

type Player struct {
	Key            PlayerKey
	Coins          int64
	DailyClaimedAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewPlayer(key PlayerKey, now time.Time) Player {
	return Player{Key: key, Coins: StartingCoins, CreatedAt: now, UpdatedAt: now}
}

func (p *Player) Debit(amount int64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}
	if p.Coins < amount {
		return ErrInsufficientCoins
	}
	p.Coins -= amount
	return nil
}

func (p *Player) Credit(amount int64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}
	p.Coins += amount
	return nil
}

var sellPrices = [MaxStars + 1]int64{SellPrice, 150, 200, 300, 500, 1000}

func SellPriceFor(stars int) int64 {
	if stars < 0 || stars > MaxStars {
		return SellPrice
	}
	return sellPrices[stars]
}
