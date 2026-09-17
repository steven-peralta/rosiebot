package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

const subHelp = "help"

func helpField(name, value string) *discordgo.MessageEmbedField {
	return &discordgo.MessageEmbedField{Name: name, Value: value}
}

func waifuHelpEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "/waifu · how it works",
		Color: brandingColor,
		Description: fmt.Sprintf("Collect waifus from MyWaifuList with coins. You start with %d coins. `/w` is a shorthand for every subcommand.",
			domain.StartingCoins),
		Fields: []*discordgo.MessageEmbedField{
			helpField("roll · "+fmt.Sprint(domain.RollCost)+" coins", fmt.Sprintf(
				"A d100 decides the pull: **1** is the Waifu of the Day, **2 to 13** is a critical roll drawn from the ranked set, anything else is a random character from the whole catalog. If you already own the result the dice are thrown again, up to %d times, and you're not charged if every attempt lands on something you own.",
				domain.MaxRerollAttempts)),
			helpField("roll banner:true · "+fmt.Sprint(domain.BannerRollCost)+" coins",
				"A roll on this week's banner: **1 to 8** gives a featured character you don't own yet, chosen uniformly, **9 to 20** is a critical roll, the rest is a regular pull. Once you own every featured character the banner slot turns into a critical roll."),
			helpField("daily · "+fmt.Sprint(domain.DailyCoins)+" coins",
				"Claim once per day. The day resets at 10:00 bot time. A d100 of **1** pays five times, **2 to 21** pays double."),
			helpField("coins · owned · sell",
				fmt.Sprintf("`coins` shows a balance. `owned` pages through a collection with a sort option; your own cards carry a Sell button. `owned view:compact` lists twenty per page with stars and rank, plus the collection's total sell value. Selling pays %s coins for an unranked waifu and more for stars: %s / %s / %s / %s / %s for one to five. You can also right-click any bot message that shows a waifu you own and pick **Sell Waifu**. `sellall` sells everything at or below a star rating in one go, after a confirmation that shows the count and payout.",
					thousands(domain.SellPrice), coins(domain.SellPriceFor(1)), coins(domain.SellPriceFor(2)), coins(domain.SellPriceFor(3)), coins(domain.SellPriceFor(4)), coins(domain.SellPriceFor(5)))),
			helpField("search · list",
				"`search` finds characters by name with autocomplete. `list` browses without a name. Both accept `series`, `sort`, `min_stars`, `min_likes`, `max_trash` and `ranked` filters, default to best rank first, and page one card at a time with a jump menu."),
			helpField("today · banner · random",
				"`today` (or `/wotd`) shows the Waifu of the Day, a ranked 1 to 4 star pick that changes at local midnight. `banner` shows this week's featured series with a Roll on banner button anyone can press; the banner rotates every Monday at 10:00 bot time. `random` shows any character."),
			helpField("trade",
				"`trade @user` opens a private builder: pick from both collections, filter by name, then Send. Adding `give` or `receive` lists sends the offer straight away. One side may be empty to make a gift. The other player can Accept, Counter with their own offer, or Decline; offers expire after ten minutes."),
			helpField("stars and rank",
				"Ratings come from `((likes+1)/(trash+1)) × (likes+trash)` over every character with more than 100 votes. The top 1% earn five stars, then 6%, 16% and 26% for four, three and two. Everyone else ranked has one star; characters under 100 votes are unranked and show no colour."),
		},
	}
}

func seriesHelpEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "/series · how it works",
		Color:       brandingColor,
		Description: "Look up an anime, game, or other series. `/s` is a shorthand.",
		Fields: []*discordgo.MessageEmbedField{
			helpField("search <query>",
				"Autocompletes series names. The card shows the cover, description, and the series' characters ranked first with their stars and rank, each linked to MyWaifuList. Press **Browse characters** to page through them one at a time."),
			helpField("filtered browsing",
				"To sort or filter a series' characters, use `/waifu list series:<name>` with the usual `sort`, `min_stars`, `min_likes`, `max_trash` and `ranked` options."),
		},
	}
}

func adminHelpEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "/admin · how it works",
		Color:       brandingColor,
		Description: "Bot owner only; server administrators cannot use these. Every reply is private and every action is written to the bot log.",
		Fields: []*discordgo.MessageEmbedField{
			helpField("coins set | increment | decrement @user <amount>",
				"Set a balance outright, add coins, or take coins. A balance never goes below zero; a decrement that would do so is refused. The player is created if they've never used the bot."),
			helpField("waifu add | remove @user <waifu>",
				"Give a character to a player or take one away. `add` autocompletes from the ranked set, `remove` from the player's collection. No coins change hands either way."),
			helpField("banner reroll",
				"Replace this week's banner with a different eligible series: at least five ranked characters and one with four or more stars, never the current or last week's series. Everyone's banner rolls switch immediately."),
			helpField("ranking status",
				"Show the current star ranking snapshot, when the next daily refresh is due, and live progress of a refresh that is running. On a fresh database the first walk starts at boot and takes about half an hour."),
		},
	}
}

func (b *Bot) help(ic *interaction, embed *discordgo.MessageEmbed) {
	err := b.s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Flags: discordgo.MessageFlagsEphemeral},
	})
	if err != nil {
		b.log.Error("help reply failed", "err", err)
	}
}
