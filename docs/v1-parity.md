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
| `!wroll` | `/waifu roll` | d100 = 1 rolls the waifu of the day | `domain.TestRollKind_D100Table`, `app.TestRollService_WaifuOfTheDayRoll` |
| | | d100 in 2..13 is a critical roll: uniform pick from the ranked set | `domain.TestRollKind_D100Table`, `domain.TestRanking_RandomAndSample`, `app.TestRollService_CriticalUsesRankedSet`, `TestRollService_CriticalFallbackBeforeSnapshot` |
| | | d100 in 14..100 is a regular roll: uniform pick from the catalog | `domain.TestRollKind_D100Table` |
| | | Already-owned result rerolls the whole d100 | `app.TestRollService_RerollOnOwned`, `TestRollService_LostRaceRerolls`, `TestRollService_RerollCapUncharged` |
| | | Balance below 200 returns `You don't have enough coins!` without an API call | `app.TestRollService_InsufficientCoinsNoAPICall`, `discord.TestRoll_Texts` |
| | | Debit and inventory insert are atomic; concurrent rolls cannot overspend | `app.TestRollService_ConcurrentDebitLosesWhenBalanceGone`, `TestRollService_DetailFailureDoesNotCharge`, `postgres.TestRepo_DebitCoins_ConcurrentDoubleSpend`, `postgres.TestServices_RollRaceOnPostgres` |
| | | Critical prefix `:sparkles: **CRITICAL ROLL!!** :sparkles:` | `discord.TestRoll_Texts` |
| | | WOTD prefix `:star2: **You rolled the Waifu of the Day. Congrats!**` | `discord.TestRoll_Texts` |
| | | Trailer `Here's who you rolled:` | `discord.TestRoll_Texts` |
| `!wdaily` | `/waifu daily` | d100 = 1 multiplies by 5 and shows `:sparkles: **CRITICAL ROLL!!** :sparkles:` | `domain.TestDailyMultiplier_D100Table` |
| | | d100 in 2..21 multiplies by 2 with the same banner | `domain.TestDailyMultiplier_D100Table`, `app.TestDailyService_Multipliers` |
| | | Success text `You claimed :coin: N coins` | `discord.TestDaily_Texts` |
| | | One claim per 10:00 window in the bot timezone | `domain.TestDailyWindowStart_AroundTen`, `TestDailyWindowStart_DST`, `TestDailyClaimAllowed`, `app.TestDailyService_ClaimOncePerWindow`, `TestDailyService_ClaimBeforeTenUsesPreviousWindow`, `postgres.TestRepo_ClaimDaily` |
| | | Repeat claim text `You've already claimed your daily for today. You can claim again in HH:MM:SS` | `discord.TestDaily_Texts`, `discord.TestErrorText_V1Strings` |
| `!wcoins [@user]` | `/waifu coins [user]` | `You have :coin: 1 coin` / `You have :coin: N coins` | `discord.TestCoins_SingularPlural` |
| | | Target form `<target> has :coin: N coins` | `discord.TestCoins_SingularPlural` |
| `!wowned [@user]` | `/waifu owned [user]` | Paginated, one waifu per page | `app.TestInventoryService_ListOrderAndEmpty`, `discord.TestOwned_EmptyAndPager`, `discord.TestOwned_TargetUser` |
| | | Empty inventory text `Specified user doesn't own any waifus.` | `discord.TestOwned_EmptyAndPager` |
| `!wsearch <q>` | `/waifu search <query>` | Paginated results, one per page | `discord.TestSearch_Texts` |
| | | No match text `Waifu was not found.` | `discord.TestSearch_Texts` |
| `!ssearch <q>` | `/series search <query>` | Best series match, then its characters by likes descending | `app.TestSearchService_SeriesSortsCharactersByLikesAcrossPages` |
| | | Header `Showing results for series <name>` | `discord.TestSeriesSearch_Texts` |
| | | No match text `Series was not found.` | `discord.TestSeriesSearch_Texts` |
| `!wrandom` | `/waifu random` | One random waifu | `app.TestSearchService_RandomFetchesDetail` |
| `!wotd` | `/waifu today` | Same waifu for the whole day across all guilds | `app.TestWotd_SameForWholeDayAndPersisted` |
| | | Picked from ranked waifus with 1..4 stars | `domain.TestWotdEligible_StarsBetween1And4`, `app.TestWotd_PickHasBetween1And4Stars` |
| | | Resets at local midnight; text `Here's the Waifu of the Day:\nRefreshes in HH:MM:SS` | `domain.TestWotdRefreshIn`, `discord.TestRandom_AndToday` |
| | | Survives a restart | `app.TestWotd_SameForWholeDayAndPersisted`, `postgres.TestDailyStore` |
| `!wtrade` | `/waifu trade <user> [give] [receive]` | Sender must own everything offered: `you don't own X` | `domain.TestValidateTrade_Matrix`, `app.TestTradeService_ProposeRejections`, `discord.TestTrade_ViolationTexts` |
| | | Target must not own anything offered: `<target> already owns X` | `domain.TestValidateTrade_Matrix`, `app.TestTradeService_ProposeRejections`, `discord.TestTrade_ViolationTexts` |
| | | Target must own everything requested: `<target> doesn't own X` | `domain.TestValidateTrade_Matrix`, `app.TestTradeService_ProposeRejections`, `discord.TestTrade_ViolationTexts` |
| | | Sender must not own anything requested: `you already own X` | `domain.TestValidateTrade_Matrix`, `app.TestTradeService_ProposeRejections`, `discord.TestTrade_ViolationTexts` |
| | | Target confirms; accept text `You accepted the trade.` | `discord.TestTrade_AcceptDeclineFlow` |
| | | Decline text `You denied the trade.` | `discord.TestTrade_AcceptDeclineFlow` |
| | | Offer is re-validated at accept time under lock | `app.TestTradeService_AcceptRevalidates`, `postgres.TestServices_TradeAcceptRaceOnPostgres`, `postgres.TestRepo_TransferAndLock` |
| | | Duplicate slugs in a list count once | `domain.TestNormaliseTrade_Dedupe` |
| sell button | Sell Waifu context menu | Confirm text `Are you sure you want to sell your <name> for 100 coins?` | `discord.TestSell_ContextMenuFlow` |
| | | Selling removes the waifu and credits 100 coins atomically | `app.TestInventoryService_Sell`, `postgres.TestRepo_Inventory`, `postgres.TestRepo_SellOwned_ConcurrentSellsOnce` |
| | | Non-owned target rejected | `app.TestInventoryService_Sell`, `discord.TestSell_ContextMenuFlow` |

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
| Unranked characters show no stars | `waifuEmbed` | `discord.TestWaifuEmbed_Sparse` |
| Ranked set is refreshed in-process by walking `/ranking/popular` to the 100-vote cutoff | v1 `scripts/scrape.ts` + `updateScoresAndTiers` | `app.TestRankingWalker_StopsAtCutoff`, `TestRankingWalker_StopsAtLastPage`, `TestRankingWalker_ResumesAfterTransientError`, `TestRankingWalker_AbandonKeepsOldTable`, `TestRankingService_RunRefreshesThenSleepsUntilNext` |
| Ranking survives a restart | (new) | `app.TestRankingService_LoadPublishesStaleSnapshot`, `postgres.TestRankingStore_SaveLoadPrune` |

## Embed

| Behaviour | Proved by |
|---|---|
| Title starts with one `:star:` per star on its own line | `discord.TestWaifuEmbed_AllFields` |
| `:underage: ` prefix on NSFW | `discord.TestWaifuEmbed_AllFields` |
| `Name - OriginalName` when original name present | `discord.TestWaifuEmbed_AllFields` |
| Title links to the MWL page; full-width image | `discord.TestWaifuEmbed_AllFields` |
| Description spoilered and truncated to 256 characters with `...` | `discord.TestWaifuEmbed_AllFields`, `discord.TestWaifuEmbed_Sparse` |
| Inline fields: Likes, Trash, Rank, Weight (kg and rounded lbs), Height (cm and floored ft/in), Bust, Hip, Waist, Origin (spoilered), Age (including 0), Blood Type, Series | `discord.TestWaifuEmbed_AllFields`, `discord.TestHeightAndWeightConversions` |
| Non-inline `Appears In` joined by `, ` | `discord.TestWaifuEmbed_AllFields` |
| Footer `rosiebot v<version> (<ms>ms)`, colour `#7752a0` | `discord.TestSeriesEmbedAndFooter`, `discord.TestWaifuEmbed_AllFields` |

## Generic error texts

| Text | Proved by |
|---|---|
| `An unexpected error occurred.` | `discord.TestErrorText_V1Strings`, `discord.TestSearch_Texts` |
| `No data was found.` | `discord.TestErrorText_V1Strings`, `discord.TestSeriesSearch_Texts` |
| `Waifu was not found.` | `discord.TestSearch_Texts` |
| `Series was not found.` | `discord.TestSeriesSearch_Texts` |
| `User was not found. (Are you sure you @'d them correctly?)` | `discord.TestTrade_ViolationTexts` |
| `The <name> command cannot be invoked from the direct messages of the bot.` | `discord.TestDMGating_PerSubcommand` |

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
| Embed padding fills inline fields to a multiple of three | v1 pushed `len % 3` blanks, which mis-padded; the intent was full rows |
| Selling a waifu that is currently shown in a live pager re-renders that pager at the same index | New behaviour enabled by the context-menu design |
| Target player row created on demand for trades | v1 failed the trade when the target had never used the bot |
| MWL client: only `/meta/random` and `/meta/daily` go through the ogen-generated client; character detail, search, work characters and rankings are hand-rolled over the same transport | The checked-in spec declares nullable fields as non-null strings (`appearances[].studio`, `release_date`) so the generated decoders reject live payloads, and it declares no `page` parameters. See `internal/adapter/mwl/source.go` |

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
