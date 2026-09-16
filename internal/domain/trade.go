package domain

import (
	"errors"
	"fmt"
	"slices"
)

var (
	ErrTradeWithSelf   = errors.New("cannot trade with yourself")
	ErrTradeEmpty      = errors.New("a trade must include at least one waifu")
	ErrTradeOverlap    = errors.New("a waifu cannot be on both sides of a trade")
	ErrTradeNotOwned   = errors.New("waifu not owned")
	ErrTradeAlreadyOwn = errors.New("waifu already owned")
)

type TradeOffer struct {
	Sender  PlayerKey
	Target  PlayerKey
	Give    []string
	Receive []string
}

type TradeSide int

const (
	TradeSideSender TradeSide = iota
	TradeSideTarget
)

type TradeViolation struct {
	Slug string
	Side TradeSide
	Err  error
}

func (v *TradeViolation) Error() string {
	who := "sender"
	if v.Side == TradeSideTarget {
		who = "target"
	}
	return fmt.Sprintf("%s: %v (%s)", who, v.Err, v.Slug)
}

func (v *TradeViolation) Unwrap() error { return v.Err }

func NormaliseTrade(offer TradeOffer) (TradeOffer, error) {
	if offer.Sender == offer.Target {
		return TradeOffer{}, ErrTradeWithSelf
	}
	offer.Give = dedupe(offer.Give)
	offer.Receive = dedupe(offer.Receive)
	if len(offer.Give) == 0 && len(offer.Receive) == 0 {
		return TradeOffer{}, ErrTradeEmpty
	}
	for _, slug := range offer.Give {
		if slices.Contains(offer.Receive, slug) {
			return TradeOffer{}, fmt.Errorf("%w: %s", ErrTradeOverlap, slug)
		}
	}
	return offer, nil
}

func ValidateTrade(offer TradeOffer, senderOwns, targetOwns []string) error {
	for _, slug := range offer.Give {
		if !slices.Contains(senderOwns, slug) {
			return &TradeViolation{Slug: slug, Side: TradeSideSender, Err: ErrTradeNotOwned}
		}
		if slices.Contains(targetOwns, slug) {
			return &TradeViolation{Slug: slug, Side: TradeSideTarget, Err: ErrTradeAlreadyOwn}
		}
	}
	for _, slug := range offer.Receive {
		if !slices.Contains(targetOwns, slug) {
			return &TradeViolation{Slug: slug, Side: TradeSideTarget, Err: ErrTradeNotOwned}
		}
		if slices.Contains(senderOwns, slug) {
			return &TradeViolation{Slug: slug, Side: TradeSideSender, Err: ErrTradeAlreadyOwn}
		}
	}
	return nil
}

func dedupe(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	slices.Sort(out)
	return slices.Compact(out)
}
