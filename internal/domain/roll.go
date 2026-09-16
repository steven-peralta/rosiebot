package domain

import "fmt"

type RollKind int

const (
	RollRegular RollKind = iota
	RollCritical
	RollWaifuOfTheDay
)

func (k RollKind) String() string {
	switch k {
	case RollRegular:
		return "regular"
	case RollCritical:
		return "critical"
	case RollWaifuOfTheDay:
		return "waifu-of-the-day"
	default:
		return fmt.Sprintf("RollKind(%d)", int(k))
	}
}

const (
	D100Min            = 1
	D100Max            = 100
	MaxRerollAttempts  = 5
	criticalRollLow    = 2
	criticalRollHigh   = 13
	waifuOfTheDayRoll  = 1
	dailyJackpotRoll   = 1
	dailyDoubleRollLow = 2
	dailyDoubleRollHi  = 21
)

func RollKindFor(d100 int) (RollKind, error) {
	if err := checkD100(d100); err != nil {
		return RollRegular, err
	}
	switch {
	case d100 == waifuOfTheDayRoll:
		return RollWaifuOfTheDay, nil
	case d100 >= criticalRollLow && d100 <= criticalRollHigh:
		return RollCritical, nil
	default:
		return RollRegular, nil
	}
}

func DailyMultiplier(d100 int) (int, error) {
	if err := checkD100(d100); err != nil {
		return 0, err
	}
	switch {
	case d100 == dailyJackpotRoll:
		return 5, nil
	case d100 >= dailyDoubleRollLow && d100 <= dailyDoubleRollHi:
		return 2, nil
	default:
		return 1, nil
	}
}

func checkD100(v int) error {
	if v < D100Min || v > D100Max {
		return fmt.Errorf("d100 out of range: %d", v)
	}
	return nil
}
