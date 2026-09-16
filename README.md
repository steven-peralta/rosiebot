# rosiebot

A Discord waifu gacha bot backed by [MyWaifuList](https://mywaifulist.moe). Roll, collect, trade, and sell waifus with slash commands.

## Commands

| Command | What it does |
|---|---|
| `/waifu roll` | Spend 200 coins on a random waifu. 12% chance of a critical roll from the ranked set, 1% chance of the waifu of the day. |
| `/waifu daily` | Claim 400 coins once per day (window resets at 10:00 bot time), with a chance to double or quintuple. |
| `/waifu coins [user]` | Show a balance. |
| `/waifu owned [user]` | Browse a collection, one waifu per page. |
| `/waifu search <query>` | Search waifus. |
| `/waifu random` | Show a random waifu. |
| `/waifu today` | Show the waifu of the day. |
| `/waifu trade <user> [give] [receive]` | Offer a trade or a gift; the other side confirms with a button. |
| `/series search <query>` | Find a series and list its waifus by likes. |
| **Sell Waifu** (message context menu) | Right-click any bot message showing a waifu you own to sell it for 100 coins. |

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
