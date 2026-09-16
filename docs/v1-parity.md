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
| | | Roll results carry a Roll again button restricted to the roller; failures (no coins) arrive as an ephemeral follow-up without touching the card | `discord.TestRoll_AgainButton` (new in v2) |
| `!wdaily` | `/waifu daily` | d100 = 1 multiplies by 5 and shows `:sparkles: **CRITICAL ROLL!!** :sparkles:` | `domain.TestDailyMultiplier_D100Table` |
| | | d100 in 2..21 multiplies by 2 with the same banner | `domain.TestDailyMultiplier_D100Table`, `app.TestDailyService_Multipliers` |
| | | Success text `You claimed :coin: N coins` | `discord.TestDaily_Texts` |
| | | One claim per 10:00 window in the bot timezone | `domain.TestDailyWindowStart_AroundTen`, `TestDailyWindowStart_DST`, `TestDailyClaimAllowed`, `app.TestDailyService_ClaimOncePerWindow`, `TestDailyService_ClaimBeforeTenUsesPreviousWindow`, `postgres.TestRepo_ClaimDaily` |
| | | Repeat claim text `You've already claimed your daily for today. You can claim again in HH:MM:SS` | `discord.TestDaily_Texts`, `discord.TestErrorText_V1Strings` |
| `!wcoins [@user]` | `/waifu coins [user]` | `You have :coin: 1 coin` / `You have :coin: N coins` | `discord.TestCoins_SingularPlural` |
| | | Target form `<target> has :coin: N coins` | `discord.TestCoins_SingularPlural` |
| `!wowned [@user]` | `/waifu owned [user]` | Paginated, one waifu per page | `app.TestInventoryService_ListOrderAndEmpty`, `discord.TestOwned_EmptyAndPager`, `discord.TestOwned_TargetUser` |
| | | Empty inventory text `Specified user doesn't own any waifus.` | `discord.TestOwned_EmptyAndPager` |
| `!wsearch <q>` | `/waifu search [query]` | Paginated results, one per page; a query-less call with at least one other option browses the in-memory ranked set (up to 500 results, rank order by default) or, before a snapshot exists, the first 30 catalog entries (v1 listed the first 100 from its database); with no input at all the bot asks for a name or an option | `discord.TestSearch_Texts`, `discord.TestSearch_SoftOptionalQuery`, `discord.TestSearch_MinStarsAloneBrowsesRankedSet`, `app.TestSearchService_EmptyTermBrowsesRankedSet`, `app.TestSearchService_EmptyTermListsCatalog` |
| | | Search covers husbandos as well as waifus (combined `/search` endpoint) | `mwl.TestSource_SearchWaifus`, live smoke `shinji` |
| | | No match text `Waifu was not found.` | `discord.TestSearch_Texts` |
| `!ssearch <q>` | `/waifu search series:<name>` | Series search is an option of waifu search: `series` autocompletes series names via the cached works search, a picked suggestion opens that series, `query` narrows its characters by name, and the sort and filter options apply; characters list most liked first by default | `app.TestSearchService_SeriesSortsCharactersByLikesAcrossPages`, `TestSearchService_SeriesBySlugAndOptions`, `TestSearchService_SeriesQueryNarrowsByName`, `TestSearchService_SuggestSeries`, `discord.TestSearch_SeriesOption`, `TestSearch_SeriesAutocomplete` |
| | | Header `Showing results for series <name>` | `discord.TestSearch_SeriesOption` |
| | | No match text `Series was not found.` | `discord.TestSearch_SeriesOption` |
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

The v2 card is a deliberate redesign (owner request during live testing on 2026-09-16). It keeps every piece of v1 information but lays it out differently.

| Behaviour | Proved by |
|---|---|
| Title is the name, prefixed with 🔞 when NSFW, linked to the MWL page | `discord.TestWaifuEmbed_FullCard` |
| First appearance shown as the author line, linked to the series | `discord.TestWaifuEmbed_FullCard` |
| Original and romaji names on an italic line under the title (deduplicated) | `discord.TestWaifuEmbed_FullCard`, `TestWaifuEmbed_SparseAndUnranked` |
| Stars rendered as ★/☆ out of five with the rank position; unranked characters read `Unranked` | `discord.TestWaifuEmbed_FullCard`, `TestWaifuEmbed_SparseAndUnranked` |
| Likes, trash and liked percentage on one line with thousands separators | `discord.TestWaifuEmbed_FullCard`, `TestThousands` |
| Description spoilered and truncated to 256 characters with `...` | `discord.TestWaifuEmbed_FullCard` |
| Vitals field: height (cm and ft/in), weight (kg and rounded lb), B·W·H | `discord.TestWaifuEmbed_FullCard`, `TestHeightAndWeightConversions` |
| Details field: age (including 0), blood type, spoilered origin | `discord.TestWaifuEmbed_FullCard` |
| Appears in: up to six series, then `+N more` | `discord.TestWaifuEmbed_AppearancesCapped` |
| Card colour follows the star tier; unranked cards have no accent colour | `discord.TestWaifuEmbed_SparseAndUnranked`, `TestSeriesEmbedAndFooter` |
| Footer `rosiebot v<version> · <ms>ms` | `discord.TestSeriesEmbedAndFooter` |
| Full-width image | `discord.TestWaifuEmbed_FullCard` |
| Owned cards viewed by their owner carry a 💰 Sell button in addition to the context menu | `discord.TestSell_ButtonOnOwnedCard`, `TestPagerComponents` |

## Generic error texts

| Text | Proved by |
|---|---|
| `An unexpected error occurred.` | `discord.TestErrorText_V1Strings`, `discord.TestSearch_Texts` |
| `No data was found.` | `discord.TestErrorText_V1Strings`, `discord.TestSearch_SeriesOption` |
| `Waifu was not found.` | `discord.TestSearch_Texts` |
| `Series was not found.` | `discord.TestSearch_SeriesOption` |
| `User was not found. (Are you sure you @'d them correctly?)` | `discord.TestTrade_ViolationTexts` |
| `The <name> command cannot be invoked from the direct messages of the bot.` | `discord.TestDMGating_PerSubcommand` |
| `/series search` folded into `/waifu search` as the `series` option | Owner decision: one search surface with a consistent UX |

## Deliberate departures from v1

| Change | Reason |
|---|---|
| d100 is 1..100 (v1's `randomInt(1, 100)` produced 1..99) | Owner's uncommitted master fix already made this change for rolls; applied to daily as well |
| Daily window is a fixed 10:00 boundary, not 10:00 on the day after the claim | Owner decision; removes the 25 hour lockout edge case |
| Trades may be one-sided gifts | Owner decision |
| Sell is a message context menu command instead of a reaction button | Owner decision; works on any bot message showing a waifu and survives restarts |
| Reroll on already-owned capped at 5 attempts, uncharged on exhaustion | v1 recursed without bound |
| v1 `sortby:`/`field:` text tokens replaced by typed slash options: `sort` (choice list), `min_stars` (1..5), `min_likes`, `max_trash`, `ranked`; sorting and filtering apply to the fetched result set (up to 30 results) | Owner asked to leverage slash command features instead of a text mini-language; the MWL search endpoint is term only (`app.TestSearchService_SortAndFilters`, `TestQuery_Apply`, `discord.TestSearch_TypedOptions`) |
| `query` autocompletes character names from the in-memory ranking table; picking a suggestion opens that card directly | `discord.TestSearch_Autocomplete`, `app.TestSearchService_Suggest` |
| Result pagers include a select menu of up to 25 results for direct jumps, alongside the v1-style buttons and jump modal | `discord.TestOwned_SortAndSelectMenu` |
| `/waifu owned` gains a `sort` option (oldest, newest, rank, likes, name); default stays acquisition order | `discord.TestOwned_SortAndSelectMenu` |
| `studio:` branch of series search dropped | Broken in v1 (crashed on no match) |
| Numeric MWL id no longer shown in the embed title | The current MWL API does not expose it on character detail |
| Waifu card redesigned: grouped Vitals/Details fields, stats line, author line, tier colours | Owner found the v1 grid of twelve emoji fields hard to read |
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
