package discord

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	msgUnexpected          = "An unexpected error occurred."
	msgNoData              = "No data was found."
	msgWaifuNotFound       = "Waifu was not found."
	msgSeriesNotFound      = "Series was not found."
	msgUserNotFound        = "User was not found. (Are you sure you @'d them correctly?)"
	msgInsufficientCoins   = "You don't have enough coins!"
	msgOwnsNothing         = "Specified user doesn't own any waifus."
	msgRollExhausted       = "Couldn't find a waifu you don't already own. Try again!"
	msgDMFmt               = "The %s command cannot be invoked from the direct messages of the bot."
	msgExpired             = "This menu has expired. Run the command again."
	msgNotYourMenu         = "This menu belongs to someone else."
	msgCritical            = ":sparkles: **CRITICAL ROLL!!** :sparkles:"
	msgWotdRoll            = ":star2: **You rolled the Waifu of the Day. Congrats!**"
	msgRolled              = "Here's who you rolled:"
	msgDailyClaimedFmt     = "You've already claimed your daily for today. You can claim again in %s"
	msgDailyFmt            = "You claimed :coin: %s coins"
	msgDailyCriticalFmt    = "%s You claimed :coin: %s coins!"
	msgWotdFmt             = "Here's the Waifu of the Day:\nRefreshes in %s"
	msgBannerRoll          = ":confetti_ball: **BANNER ROLL!!** :confetti_ball:"
	msgBannerFmt           = "Here's this week's banner:\nRefreshes in %s"
	msgNoBanner            = "There's no banner this week yet. Check back soon!"
	msgSeriesHeaderFmt     = "Showing results for series %s"
	msgSeriesFound         = "Here's the series I found:"
	msgAdminOnly           = "Only the bot owner can use this command."
	msgAdminSetFmt         = "Set %s's balance to :coin: %s coins."
	msgAdminAddedFmt       = "Added :coin: %s coins to %s. New balance: %s coins."
	msgAdminRemovedFmt     = "Removed :coin: %s coins from %s. New balance: %s coins."
	msgAdminGrantedFmt     = "Gave %s to %s."
	msgAdminRevokedFmt     = "Took %s from %s."
	msgAdminAlreadyOwnsFmt = "%s already owns that waifu."
	msgAdminNotOwnedFmt    = "%s doesn't own that waifu."
	msgAdminBelowZeroFmt   = "%s doesn't have that many coins."
	msgTradeAccepted       = "You accepted the trade."
	msgTradeDenied         = "You denied the trade."
	msgTradeConflict       = "That trade is no longer valid; someone's collection changed."
	msgTradeExpired        = "This trade request has expired."
	msgTradeOfferFmt       = "%s: %s is offering the following trade request:"
	msgSellConfirmFmt      = "Are you sure you want to sell your %s for %s coins?"
	msgSoldFmt             = "Sold %s for %s coins. You now have %s coins."
	msgNotOwnedFmt         = "You don't own %s."
	msgNotAWaifu           = "That message doesn't show a waifu."
	msgSellCancelled       = "Sale cancelled."
	msgRateLimited         = "MyWaifuList is rate limiting me right now. Try again in a minute."
	msgNotYourRoll         = "That button belongs to someone else's roll. Use /waifu roll to roll for yourself."
	msgFilteredOutFmt      = "%d results matched, but none passed your filters."
	msgPickFilteredFmt     = "%s doesn't pass your filters (%s)."
)

func mention(id string) string {
	return "<@" + id + ">"
}

func (b *Bot) deferReply(ic *interaction, ephemeral bool) bool {
	resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource}
	if ephemeral {
		resp.Data = &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}
	}
	if err := b.s.InteractionRespond(ic.Interaction, resp); err != nil {
		b.log.Error("defer failed", "err", err)
		return false
	}
	return true
}

func (b *Bot) replyEphemeral(ic *interaction, content string) {
	err := b.s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
	})
	if err != nil {
		b.log.Error("ephemeral reply failed", "err", err)
	}
}

func (b *Bot) replyEphemeralComponents(ic *interaction, content string, components []discordgo.MessageComponent) {
	err := b.s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral, Components: components},
	})
	if err != nil {
		b.log.Error("ephemeral reply failed", "err", err)
	}
}

func (b *Bot) updateMessage(ic *interaction, content string, embeds []*discordgo.MessageEmbed, components []discordgo.MessageComponent) {
	if embeds == nil {
		embeds = []*discordgo.MessageEmbed{}
	}
	if components == nil {
		components = []discordgo.MessageComponent{}
	}
	if oversized(content, embeds) {
		b.log.Warn("response too large", "user", ic.userID(), "content_runes", runes(content), "embeds", len(embeds))
		b.replyEphemeral(ic, msgResponseTooLarge)
		return
	}
	err := b.s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Content: content, Embeds: embeds, Components: components},
	})
	if err != nil {
		b.log.Error("update message failed", "err", err)
	}
}

func (b *Bot) edit(ic *interaction, content string, embeds []*discordgo.MessageEmbed, components []discordgo.MessageComponent) *discordgo.Message {
	if embeds == nil {
		embeds = []*discordgo.MessageEmbed{}
	}
	if components == nil {
		components = []discordgo.MessageComponent{}
	}
	if oversized(content, embeds) {
		b.log.Warn("response too large", "user", ic.userID(), "content_runes", runes(content), "embeds", len(embeds))
		content = mention(ic.userID()) + " " + msgResponseTooLarge
		embeds = []*discordgo.MessageEmbed{}
		components = []discordgo.MessageComponent{}
	}
	msg, err := b.s.InteractionResponseEdit(ic.Interaction, &discordgo.WebhookEdit{Content: &content, Embeds: &embeds, Components: &components})
	if err != nil {
		b.log.Error("edit response failed", "err", err)
		if len(embeds) > 0 || len(components) > 0 {
			fallback := mention(ic.userID()) + " " + msgUnexpected
			empty := []*discordgo.MessageEmbed{}
			none := []discordgo.MessageComponent{}
			if _, ferr := b.s.InteractionResponseEdit(ic.Interaction, &discordgo.WebhookEdit{Content: &fallback, Embeds: &empty, Components: &none}); ferr != nil {
				b.log.Error("fallback edit failed", "err", ferr)
			}
		}
		return nil
	}
	return msg
}

func (b *Bot) editText(ic *interaction, content string) {
	b.edit(ic, content, nil, nil)
}

func (b *Bot) editChannelMessage(channelID, messageID, content string, embeds []*discordgo.MessageEmbed, components []discordgo.MessageComponent) {
	if embeds == nil {
		embeds = []*discordgo.MessageEmbed{}
	}
	if components == nil {
		components = []discordgo.MessageComponent{}
	}
	_, err := b.s.ChannelMessageEditComplex(&discordgo.MessageEdit{Channel: channelID, ID: messageID, Content: &content, Embeds: &embeds, Components: &components})
	if err != nil {
		b.log.Warn("channel message edit failed", "channel", channelID, "message", messageID, "err", err)
	}
}

func (b *Bot) failed(ic *interaction, action string, err error) {
	b.log.Error(action+" failed", "user", ic.userID(), "guild", ic.GuildID, "err", err)
	b.editText(ic, mention(ic.userID())+" "+errorText(err))
}

func errorText(err error) string {
	var daily *app.DailyAlreadyClaimedError
	var violation *domain.TradeViolation
	switch {
	case errors.Is(err, app.ErrInsufficientCoins):
		return msgInsufficientCoins
	case errors.Is(err, app.ErrRollExhausted):
		return msgRollExhausted
	case errors.As(err, &daily):
		return fmt.Sprintf(msgDailyClaimedFmt, domain.FormatCountdown(daily.RefreshIn))
	case errors.As(err, &violation):
		return violationText(violation, "", "")
	case errors.Is(err, app.ErrTradeConflict):
		return msgTradeConflict
	case errors.Is(err, domain.ErrTradeWithSelf):
		return "You can't trade with yourself."
	case errors.Is(err, domain.ErrTradeEmpty):
		return "A trade needs at least one waifu on either side."
	case errors.Is(err, domain.ErrTradeOverlap):
		return "A waifu can't be on both sides of a trade."
	case errors.Is(err, app.ErrRateLimited):
		return msgRateLimited
	case errors.Is(err, app.ErrNoBanner):
		return msgNoBanner
	case errors.Is(err, app.ErrNoRanking):
		return msgNoRankingYet
	case errors.Is(err, app.ErrNotFound):
		return msgNoData
	default:
		return msgUnexpected
	}
}

func violationText(v *domain.TradeViolation, targetMention, name string) string {
	if name == "" {
		name = v.Slug
	}
	if targetMention == "" {
		targetMention = "they"
	}
	switch {
	case v.Side == domain.TradeSideSender && errors.Is(v.Err, domain.ErrTradeNotOwned):
		return fmt.Sprintf("you don't own %s", name)
	case v.Side == domain.TradeSideSender:
		return fmt.Sprintf("you already own %s", name)
	case errors.Is(v.Err, domain.ErrTradeNotOwned):
		return fmt.Sprintf("%s doesn't own %s", targetMention, name)
	default:
		return fmt.Sprintf("%s already owns %s", targetMention, name)
	}
}

const (
	msgAdminRerolledFmt = "This week's banner is now **%s**."
	msgAdminNoRanking   = "The ranking isn't loaded yet, so no banner can be picked. Try again in a minute."
	msgAdminNoSeries    = "Couldn't find an eligible series this time. Try again."
	msgNoRankingYet     = "The ranking isn't loaded yet. Try again in a minute."
	msgResponseTooLarge = "Response too large!"

	msgRankingStatusUnavailable = "Ranking status isn't available in this deployment."
	msgNoOwner                  = "No bot owner is configured, so admin commands are disabled."
	msgSellAllNone              = "You have no waifus at or below that rating."
	msgSellAllConfirmFmt        = "Sell **%s** waifus (%s) for :coin: %s coins? This can't be undone."
	msgSellAllDoneFmt           = "Sold %s waifus for :coin: %s coins. You now have %s coins."
	msgSellAllViewing           = "These are the waifus that would be sold:"
)
