# rosiebot

A Discord waifu gacha bot backed by [MyWaifuList](https://mywaifulist.moe). Roll, collect, trade, and sell waifus with slash commands.

## Commands

| Command | What it does |
|---|---|
| `/waifu roll` | Spend 200 coins on a random waifu. 12% chance of a critical roll from the ranked set, 1% chance of the waifu of the day. The result carries a Roll again button. |
| `/waifu daily` | Claim 400 coins once per day (window resets at 10:00 bot time), with a chance to double or quintuple. |
| `/waifu coins [user]` | Show a balance. |
| `/waifu owned [user] [sort]` | Browse a collection, one waifu per page, with a Sell button on your own. |
| `/waifu search <query> [series] [sort] [min_stars] [min_likes] [max_trash] [ranked]` | Search waifus by name with autocomplete; `series` (also autocompleted) limits the search to one series. Results default to best rank first, unranked ones alphabetically. |
| `/waifu list [series] [sort] [min_stars] [min_likes] [max_trash] [ranked]` | Browse without a name: the ranked set (every character with more than 100 votes) by rank, or one series, filtered and sorted the same way as search. |
| `/waifu random` | Show a random waifu. |
| `/waifu today` | Show the waifu of the day. |
| `/waifu trade <user> [give] [receive]` | Offer a trade or a gift; the other side confirms with a button. |
| `/w …` | Shorthand for every `/waifu` subcommand. |
| **Sell Waifu** (message context menu) | Right-click any bot message showing a waifu you own to sell it for 100 coins. |

Character details and search pages are cached in Postgres and refreshed lazily: a stale entry is served immediately while one background request refreshes it, so nothing depends on a scheduled job. Random rolls and the waifu of the day always go to the live API, so newly submitted characters appear as soon as MyWaifuList lists them.

Star ratings come from the owner's formula, `((likes+1)/(trash+1)) * (likes+trash)`, computed over every character with more than 100 votes and refreshed in-process once a day.

## Configuration

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `DISCORD_TOKEN` | yes | | Bot token (`DISCORD_TOKEN_KEY` is accepted for v1 compatibility) |
| `WAIFU_API_KEY` | yes | | MyWaifuList API key |
| `DATABASE_URL` | yes | | Postgres connection string |
| `BOT_TIMEZONE` | no | `America/Chicago` | Timezone for daily and waifu-of-the-day resets |
| `DEV_GUILD_ID` | no | | Register commands to one guild (instant) instead of globally |
| `RANKING_REFRESH` | no | `24h` | How often the ranked set is rebuilt |
| `RANKING_MIN_VOTES` | no | `100` | Vote threshold for the ranked set |
| `WAIFU_CACHE_TTL` | no | `24h` | How long a character's detail is served from the cache before a background refresh |
| `SEARCH_CACHE_TTL` | no | `1h` | How long search, catalog and series pages are served from the cache |
| `MIGRATE_ON_START` | no | `true` | Run database migrations on boot |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, or `error` |

## Development

Requires Go 1.27 (the toolchain downloads automatically), [Task](https://taskfile.dev), and Docker for the Postgres tests.

```sh
task check          # generate freshness, lint, tests with coverage gates, build
task test:db        # Postgres adapter tests against a disposable container
task generate       # ogen client, sqlc queries, mockery mocks
task migrate:new -- add_thing
task run -- --dry-run
```

`docs/v1-parity.md` maps every behaviour of the original TypeScript bot to the tests that prove it.
