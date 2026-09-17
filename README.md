# rosiebot

A Discord waifu gacha bot backed by [MyWaifuList](https://mywaifulist.moe). Roll, collect, trade, and sell waifus with slash commands.

## Commands

| Command | What it does |
|---|---|
| `/waifu roll` | Spend 200 coins on a random waifu. 12% chance of a critical roll from the ranked set, 1% chance of the waifu of the day. The result shows your remaining balance and carries Roll again and Sell buttons. |
| `/waifu daily` | Claim 400 coins once per day (window resets at 10:00 bot time), with a chance to double or quintuple. |
| `/waifu coins [user]` | Show a balance. |
| `/waifu owned [user] [sort] [view] [series]` | Browse a collection, one waifu per page with a Sell button on your own, or `view:compact` for twenty per page with stars, rank, and the collection's total sell value. `series` narrows it to one series and shows how many of its characters you have. |
| `/waifu history [user]` | The last ten rolls with rating, roll type and when. |
| `/profile [user]` | Coins, collection size and value, rating breakdown, rarest pull, roll count, favorites, and server rank. |
| `/leaderboard [by]` | Top ten in the server by collection value (default), coins, collection size, or 4-star and 5-star count, with your own rank if you are outside the top ten. |
| `/waifu sellall <max_stars>` | Sell every waifu you own at or below a star rating, unranked included. The private confirmation lists them compact-style with paging, a View characters button that opens the card pager, and the payout. |
| `/waifu search <query> [series] [sort] [min_stars] [min_likes] [max_trash] [ranked]` | Search waifus by name with autocomplete; `series` (also autocompleted) limits the search to one series. Results default to best rank first, unranked ones alphabetically. |
| `/waifu list [series] [sort] [min_stars] [min_likes] [max_trash] [ranked]` | Browse without a name: the ranked set (every character with more than 100 votes) by rank, or one series, filtered and sorted the same way as search. |
| `/waifu random` | Show a random waifu. |
| `/waifu today` | Show the waifu of the day: a ranked 1 to 4 star pick that changes at local midnight. |
| `/wotd` | Shorthand for `/waifu today`. |
| `/favs waifus [user] [view]`, `/favs series [user] [view]` | List your favorite waifus or series, or another user's. Every waifu card and series card has a 🤍 Favorite button, shown as 💔 Unfavorite once you have it. |
| `/favs alerts on\|off` | Private DM alerts, on by default: a favorite waifu or series on the weekly banner, a favorite as Waifu of the Day, or someone else in the server rolling a favorite. Never posted in channels; one DM per event; stops automatically if your DMs are closed. |
| `/waifu banner` | Show this week's banner: a featured series, its ranked characters, and a Roll on banner button anyone can press. |
| `/waifu roll banner:true` | Spend 400 coins on a banner roll: 8% chance of a featured character you don't own yet, 12% critical, 80% regular. Once you own every featured character the banner slot becomes a critical roll. |
| `/waifu trade <user> [give] [receive]` | Offer a trade or a gift. With no `give` or `receive`, a private trade builder opens: pick from both collections with paged menus, filter by name, then Send. The other side can Accept, Counter (opens the builder prefilled with the reversed offer), or Decline. |
| `/w …` | Shorthand for every `/waifu` subcommand. |
| `/waifu help`, `/series help`, `/admin help` | Private explainer of each command's mechanics: odds, costs, filters, trading, stars, and the admin tools. |
| `/series search <query>` | Show a series card: cover, description, and its characters ranked first with stars, and a Browse characters button that opens the usual one-per-page pager. Autocompletes series names. |
| `/s …` | Shorthand for `/series`. |
| `/admin coins set\|increment\|decrement <user> <amount>` | Bot owner only: set, add to, or take from a player's balance. Replies privately. |
| `/admin waifu add\|remove <user> <waifu>` | Bot owner only: give a waifu to a player or take one away, with autocomplete. No coins change hands. |
| `/admin banner reroll` | Bot owner only: replace this week's banner with a different eligible series, never the current or last week's one. |
| `/admin ranking status` | Bot owner only: show the star ranking snapshot, when the next refresh is due, and the live progress of a running refresh. |
| `/admin ranking refresh` | Bot owner only: start a ranking walk now instead of waiting for the daily one. |
| **Sell Waifu** (message context menu) | Right-click any bot message showing a waifu you own to sell it. Unranked waifus pay 100 coins; ranked ones pay 150, 200, 300, 500 or 1000 for one to five stars. |

Character details and search pages are cached in Postgres and refreshed lazily: a stale entry is served immediately while one background request refreshes it, so nothing depends on a scheduled job. Random rolls and the waifu of the day always go to the live API, so newly submitted characters appear as soon as MyWaifuList lists them.

Every Monday at 10:00 bot time a new banner series is picked at random from series with at least five ranked characters and at least one 4-star, never the same series two weeks in a row. The pick is persisted, so restarts keep it.

Star ratings come from a like-to-trash ratio weighted by vote volume, computed over every character with more than 100 votes and refreshed in-process once a day. The like share is shrunk toward the site-wide average with a prior of 100 votes before the ratio is taken, so small samples with a lucky ratio do not outrank well-liked characters with thousands of votes: `p = (likes + 100·avg) / (votes + 100)`, `score = p / (1 − p) × votes`. On a fresh database the first walk starts as soon as the bot boots and takes roughly half an hour at the background request rate; `/admin ranking status` shows its progress.

## Configuration

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `DISCORD_TOKEN` | yes | | Bot token (`DISCORD_TOKEN_KEY` is accepted for v1 compatibility) |
| `WAIFU_API_KEY` | yes | | MyWaifuList API key |
| `DATABASE_URL` | yes | | Postgres connection string |
| `BOT_TIMEZONE` | no | `America/Chicago` | Timezone for daily and waifu-of-the-day resets and the weekly banner rotation (Monday 10:00) |
| `DEV_GUILD_ID` | no | | Register commands to one guild (instant) instead of globally |
| `BOT_OWNER_IDS` | no | | Comma-separated Discord user ids allowed to use `/admin`. Empty disables the admin commands. |
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
