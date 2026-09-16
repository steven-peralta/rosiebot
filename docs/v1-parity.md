# v1 parity checklist

Source of truth for v1 behaviour is branch `master` (TypeScript). Every row must
name the automated test that proves it before the owning phase is complete.
Deliberate departures from v1 are listed at the bottom with their reasons.

## Constants

| Behaviour | v1 source | Proved by |
|---|---|---|
| Starting balance 200 coins | `src/db/models/User.ts` | `domain.TestConstants_MatchV1`, `TestNewPlayer_StartsWithV1Balance` |
| Roll costs 200 coins | `src/config.ts` `rollCost` | `domain.TestConstants_MatchV1` |
| Selling credits 100 coins | `src/config.ts` `sellCost` | `domain.TestConstants_MatchV1` |
| Daily grants 400 coins | `src/config.ts` `daily` | `domain.TestConstants_MatchV1` |

## Commands

| v1 | v2 | Behaviour | Proved by |
|---|---|---|---|
| `!wroll` | `/waifu roll` | d100 = 1 rolls the waifu of the day | `domain.TestRollKind_D100Table` |
| | | d100 in 2..13 is a critical roll: uniform pick from the ranked set | `domain.TestRollKind_D100Table`, `domain.TestRanking_RandomAndSample` |
| | | d100 in 14..100 is a regular roll: uniform pick from the catalog | `domain.TestRollKind_D100Table` |
| | | Already-owned result rerolls the whole d100 | |
| | | Balance below 200 returns `You don't have enough coins!` without an API call | |
| | | Debit and inventory insert are atomic; concurrent rolls cannot overspend | |
| | | Critical prefix `:sparkles: **CRITICAL ROLL!!** :sparkles:` | |
| | | WOTD prefix `:star2: **You rolled the Waifu of the Day. Congrats!**` | |
| | | Trailer `Here's who you rolled:` | |
| `!wdaily` | `/waifu daily` | d100 = 1 multiplies by 5 and shows `:sparkles: **CRITICAL ROLL!!** :sparkles:` | `domain.TestDailyMultiplier_D100Table` |
| | | d100 in 2..21 multiplies by 2 with the same banner | `domain.TestDailyMultiplier_D100Table` |
| | | Success text `You claimed :coin: N coins` | |
| | | One claim per 10:00 window in the bot timezone | `domain.TestDailyWindowStart_AroundTen`, `TestDailyWindowStart_DST`, `TestDailyClaimAllowed` |
| | | Repeat claim text `You've already claimed your daily for today. You can claim again in HH:MM:SS` | |
| `!wcoins [@user]` | `/waifu coins [user]` | `You have :coin: 1 coin` / `You have :coin: N coins` | |
| | | Target form `<target> has :coin: N coins` | |
| `!wowned [@user]` | `/waifu owned [user]` | Paginated, one waifu per page | |
| | | Empty inventory text `Specified user doesn't own any waifus.` | |
| `!wsearch <q>` | `/waifu search <query>` | Paginated results, one per page | |
| | | No match text `Waifu was not found.` | |
| `!ssearch <q>` | `/series search <query>` | Best series match, then its characters by likes descending | |
| | | Header `Showing results for series <name>` | |
| | | No match text `Series was not found.` | |
| `!wrandom` | `/waifu random` | One random waifu | |
| `!wotd` | `/waifu today` | Same waifu for the whole day across all guilds | |
| | | Picked from ranked waifus with 1..4 stars | `domain.TestWotdEligible_StarsBetween1And4` |
| | | Resets at local midnight; text `Here's the Waifu of the Day:\nRefreshes in HH:MM:SS` | |
| | | Survives a restart | |
| `!wtrade` | `/waifu trade <user> [give] [receive]` | Sender must own everything offered: `you don't own X` | |
| | | Target must not own anything offered: `<target> already owns X` | |
| | | Target must own everything requested: `<target> doesn't own X` | |
| | | Sender must not own anything requested: `you already own X` | |
| | | Target confirms; accept text `You accepted the trade.` | |
| | | Decline text `You denied the trade.` | |
| | | Offer is re-validated at accept time under lock | |
| | | Duplicate slugs in a list count once | `domain.TestNormaliseTrade_Dedupe` |
| sell button | Sell Waifu context menu | Confirm text `Are you sure you want to sell your <name> for 100 coins?` | |
| | | Selling removes the waifu and credits 100 coins atomically | |
| | | Non-owned target rejected | |

## Ranking and stars

| Behaviour | v1 source | Proved by |
|---|---|---|
| score = ((likes+1)/(trash+1)) * (likes+trash) | `Waifu.updateScoresAndTiers` | `domain.TestScore_Formula` |
| Only characters with more than 100 total votes are ranked | same | `domain.TestBuildRanking_FiltersUnderMinAndDedupes` |
| Top 1% of ranked = 5 stars | `getTier` | `domain.TestStars_PositionBoundaries` |
| Up to 6% = 4 stars | | `domain.TestStars_PositionBoundaries` |
| Up to 16% = 3 stars | | `domain.TestStars_PositionBoundaries` |
| Up to 26% = 2 stars | | `domain.TestStars_PositionBoundaries` |
| Remainder = 1 star | | `domain.TestStars_PositionBoundaries` |
| Unranked characters show no stars | `waifuEmbed` | |

## Embed

| Behaviour | Proved by |
|---|---|
| Title starts with one `:star:` per star on its own line | |
| `:underage: ` prefix on NSFW | |
| `Name - OriginalName` when original name present | |
| Title links to the MWL page; full-width image | |
| Description spoilered and truncated to 256 characters with `...` | |
| Inline fields: Likes, Trash, Rank, Weight (kg and rounded lbs), Height (cm and floored ft/in), Bust, Hip, Waist, Origin (spoilered), Age (including 0), Blood Type, Series | |
| Non-inline `Appears In` joined by `, ` | |
| Footer `rosiebot v<version> (<ms>ms)`, colour `#7752a0` | |

## Generic error texts

| Text | Proved by |
|---|---|
| `An unexpected error occurred.` | |
| `No data was found.` | |
| `Waifu was not found.` | |
| `Series was not found.` | |
| `User was not found. (Are you sure you @'d them correctly?)` | |
| `The <name> command cannot be invoked from the direct messages of the bot.` | |

## Deliberate departures from v1

| Change | Reason |
|---|---|
| d100 is 1..100 (v1's `randomInt(1, 100)` produced 1..99) | Owner's uncommitted master fix already made this change for rolls; applied to daily as well |
| Daily window is a fixed 10:00 boundary, not 10:00 on the day after the claim | Owner decision; removes the 25 hour lockout edge case |
| Trades may be one-sided gifts | Owner decision |
| Sell is a message context menu command instead of a reaction button | Owner decision; works on any bot message showing a waifu and survives restarts |
| Reroll on already-owned capped at 5 attempts, uncharged on exhaustion | v1 recursed without bound |
| `field:value` and `sortby:` search filters dropped | They were MongoDB queries over a scraped catalog; the MWL search endpoint is term only |
| `studio:` branch of series search dropped | Broken in v1 (crashed on no match) |
| Numeric MWL id no longer shown in the embed title | The current MWL API does not expose it on character detail |
| Target player row created on demand for trades | v1 failed the trade when the target had never used the bot |

## Manual release checklist (dev guild)

- [ ] `/waifu roll` regular, critical (stars visible), and WOTD outcomes
- [ ] `/waifu roll` with insufficient coins
- [ ] `/waifu daily` first claim, repeat claim countdown, and a critical multiplier
- [ ] `/waifu coins` self and target
- [ ] `/waifu owned` pagination: first, prev, next, last, jump modal, expired menu
- [ ] `/waifu search` with results and with none
- [ ] `/series search` with results and with none
- [ ] `/waifu random`
- [ ] `/waifu today` twice in a row and after a restart
- [ ] `/waifu trade` accept, decline, gift, and a conflict after the counterparty sells
- [ ] Sell Waifu on an owned page, a roll result, a non-owned search result, and a non-bot message
- [ ] DM gating for roll, daily, coins, owned, trade; search, random, today, series work in DMs
- [ ] Ranking load log line with row count and cutoff page
