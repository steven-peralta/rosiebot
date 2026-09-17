package domain

import "testing"

func TestRollKind_D100Table(t *testing.T) {
	counts := map[RollKind]int{}
	for d := D100Min; d <= D100Max; d++ {
		kind, err := RollKindFor(d)
		if err != nil {
			t.Fatalf("RollKindFor(%d): %v", d, err)
		}
		counts[kind]++
		switch {
		case d == 1 && kind != RollWaifuOfTheDay:
			t.Errorf("d=%d kind=%v, want waifu of the day", d, kind)
		case d >= 2 && d <= 13 && kind != RollCritical:
			t.Errorf("d=%d kind=%v, want critical", d, kind)
		case d >= 14 && kind != RollRegular:
			t.Errorf("d=%d kind=%v, want regular", d, kind)
		}
	}
	if counts[RollWaifuOfTheDay] != 1 || counts[RollCritical] != 12 || counts[RollRegular] != 87 {
		t.Errorf("odds = %v, want 1/12/87", counts)
	}
}

func TestRollKind_RejectsOutOfRange(t *testing.T) {
	for _, d := range []int{0, 101, -5} {
		if _, err := RollKindFor(d); err == nil {
			t.Errorf("RollKindFor(%d) should fail", d)
		}
	}
}

func TestBannerRollKind_D100Table(t *testing.T) {
	counts := map[RollKind]int{}
	for d := D100Min; d <= D100Max; d++ {
		kind, err := BannerRollKindFor(d)
		if err != nil {
			t.Fatalf("BannerRollKindFor(%d): %v", d, err)
		}
		counts[kind]++
		switch {
		case d <= 8 && kind != RollBanner:
			t.Errorf("d=%d kind=%v, want banner", d, kind)
		case d >= 9 && d <= 20 && kind != RollCritical:
			t.Errorf("d=%d kind=%v, want critical", d, kind)
		case d >= 21 && kind != RollRegular:
			t.Errorf("d=%d kind=%v, want regular", d, kind)
		}
	}
	if counts[RollBanner] != 8 || counts[RollCritical] != 12 || counts[RollRegular] != 80 || counts[RollWaifuOfTheDay] != 0 {
		t.Errorf("odds = %v, want 8/12/80 and never the waifu of the day", counts)
	}
	for _, d := range []int{0, 101} {
		if _, err := BannerRollKindFor(d); err == nil {
			t.Errorf("BannerRollKindFor(%d) should fail", d)
		}
	}
}

func TestRollKind_String(t *testing.T) {
	cases := map[RollKind]string{RollRegular: "regular", RollCritical: "critical", RollWaifuOfTheDay: "waifu-of-the-day", RollBanner: "banner", RollKind(9): "RollKind(9)"}
	for k, want := range cases {
		if k.String() != want {
			t.Errorf("%d.String() = %q, want %q", int(k), k.String(), want)
		}
	}
}

func TestDailyMultiplier_D100Table(t *testing.T) {
	counts := map[int]int{}
	for d := D100Min; d <= D100Max; d++ {
		m, err := DailyMultiplier(d)
		if err != nil {
			t.Fatalf("DailyMultiplier(%d): %v", d, err)
		}
		counts[m]++
		switch {
		case d == 1 && m != 5:
			t.Errorf("d=%d multiplier=%d, want 5", d, m)
		case d >= 2 && d <= 21 && m != 2:
			t.Errorf("d=%d multiplier=%d, want 2", d, m)
		case d >= 22 && m != 1:
			t.Errorf("d=%d multiplier=%d, want 1", d, m)
		}
	}
	if counts[5] != 1 || counts[2] != 20 || counts[1] != 79 {
		t.Errorf("odds = %v, want 1/20/79", counts)
	}
	if _, err := DailyMultiplier(0); err == nil {
		t.Error("DailyMultiplier(0) should fail")
	}
}

func TestConstants_MatchV1(t *testing.T) {
	if StartingCoins != 200 || RollCost != 200 || SellPrice != 100 || DailyCoins != 400 {
		t.Errorf("economy constants drifted from v1: start=%d roll=%d sell=%d daily=%d", StartingCoins, RollCost, SellPrice, DailyCoins)
	}
	if MaxRerollAttempts != 5 {
		t.Errorf("reroll cap = %d, want 5", MaxRerollAttempts)
	}
}
