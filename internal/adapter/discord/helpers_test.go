package discord

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/mock"

	"github.com/steven-peralta/rosiebot/internal/adapter/memory"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/app/mocks"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	guildID   = "guild-1"
	channelID = "chan-1"
	aliceID   = "alice"
	bobID     = "bob"
	botID     = "bot-user"
)

type call struct {
	kind     string
	resp     *discordgo.InteractionResponse
	edit     *discordgo.WebhookEdit
	msgEdit  *discordgo.MessageEdit
	followup *discordgo.WebhookParams
}

type fakeAPI struct {
	mu           sync.Mutex
	calls        []call
	messages     map[string]*discordgo.Message
	failEdit     bool
	failRichEdit bool
	failFetch    bool
}

func newFakeAPI() *fakeAPI {
	return &fakeAPI{messages: map[string]*discordgo.Message{}}
}

func (f *fakeAPI) InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call{kind: "respond", resp: r})
	if r.Type == discordgo.InteractionResponseUpdateMessage && i.Message != nil && r.Data != nil {
		if msg, ok := f.messages[i.Message.ID]; ok {
			msg.Content = r.Data.Content
			msg.Embeds = r.Data.Embeds
			msg.Components = r.Data.Components
		}
	}
	return nil
}

func (f *fakeAPI) InteractionResponseEdit(i *discordgo.Interaction, e *discordgo.WebhookEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call{kind: "edit", edit: e})
	if f.failEdit {
		return nil, errors.New("edit failed")
	}
	if f.failRichEdit && ((e.Embeds != nil && len(*e.Embeds) > 0) || (e.Components != nil && len(*e.Components) > 0)) {
		return nil, errors.New("HTTP 400 Bad Request")
	}
	id := "msg-" + i.ID
	msg := &discordgo.Message{ID: id, ChannelID: i.ChannelID, Author: &discordgo.User{ID: botID}}
	if e.Content != nil {
		msg.Content = *e.Content
	}
	if e.Embeds != nil {
		msg.Embeds = *e.Embeds
	}
	if e.Components != nil {
		msg.Components = *e.Components
	}
	f.messages[id] = msg
	return msg, nil
}

func (f *fakeAPI) FollowupMessageCreate(i *discordgo.Interaction, _ bool, p *discordgo.WebhookParams, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call{kind: "followup", followup: p})
	id := "fu-" + i.ID
	msg := &discordgo.Message{ID: id, ChannelID: i.ChannelID, Author: &discordgo.User{ID: botID}, Content: p.Content, Embeds: p.Embeds, Components: p.Components}
	f.messages[id] = msg
	return msg, nil
}

func (f *fakeAPI) lastFollowup() *discordgo.WebhookParams {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.calls) - 1; i >= 0; i-- {
		if f.calls[i].kind == "followup" {
			return f.calls[i].followup
		}
	}
	return nil
}

func (f *fakeAPI) ChannelMessage(_, messageID string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failFetch {
		return nil, errors.New("fetch failed")
	}
	msg, ok := f.messages[messageID]
	if !ok {
		return nil, errors.New("unknown message")
	}
	return msg, nil
}

func (f *fakeAPI) ChannelMessageEditComplex(m *discordgo.MessageEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call{kind: "msgedit", msgEdit: m})
	msg, ok := f.messages[m.ID]
	if !ok {
		msg = &discordgo.Message{ID: m.ID, ChannelID: m.Channel, Author: &discordgo.User{ID: botID}}
		f.messages[m.ID] = msg
	}
	if m.Content != nil {
		msg.Content = *m.Content
	}
	if m.Embeds != nil {
		msg.Embeds = *m.Embeds
	}
	if m.Components != nil {
		msg.Components = *m.Components
	}
	return msg, nil
}

func (f *fakeAPI) ApplicationCommandBulkOverwrite(_, _ string, cmds []*discordgo.ApplicationCommand, _ ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call{kind: "register"})
	return cmds, nil
}

func (f *fakeAPI) last() call {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[len(f.calls)-1]
}

func (f *fakeAPI) lastRespond() *discordgo.InteractionResponse {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.calls) - 1; i >= 0; i-- {
		if f.calls[i].kind == "respond" {
			return f.calls[i].resp
		}
	}
	return nil
}

func (f *fakeAPI) lastEdit() *discordgo.WebhookEdit {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.calls) - 1; i >= 0; i-- {
		if f.calls[i].kind == "edit" {
			return f.calls[i].edit
		}
	}
	return nil
}

func (f *fakeAPI) lastMsgEdit() *discordgo.MessageEdit {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.calls) - 1; i >= 0; i-- {
		if f.calls[i].kind == "msgedit" {
			return f.calls[i].msgEdit
		}
	}
	return nil
}

func (f *fakeAPI) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = nil
}

func editContent(e *discordgo.WebhookEdit) string {
	if e == nil || e.Content == nil {
		return ""
	}
	return *e.Content
}

type fakeStatus struct{ st app.RankingStatus }

func (f *fakeStatus) Status() app.RankingStatus { return f.st }

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

type seqRandom struct{ vals []int }

func (r *seqRandom) IntN(n int) int {
	if len(r.vals) == 0 {
		return 0
	}
	v := r.vals[0]
	r.vals = r.vals[1:]
	if v >= n {
		v = n - 1
	}
	return v
}

type fixture struct {
	t       *testing.T
	api     *fakeAPI
	bot     *Bot
	clock   *fakeClock
	rng     *seqRandom
	players *memory.PlayerStore
	source  *mocks.WaifuSource
	ranking *memory.RankingHolder
	daily   *memory.DailyStore
	banners *memory.BannerStore
	loc     *time.Location
	status  *fakeStatus
	seq     int
}

func summary(slug string) domain.WaifuSummary {
	return domain.WaifuSummary{Slug: slug, UUID: "u-" + slug, Name: "Name " + slug, PictureURL: "https://img/" + slug, Likes: 10, Trash: 1}
}

func detail(slug string) domain.Waifu {
	return domain.Waifu{WaifuSummary: summary(slug), URL: waifuURL(slug), Description: "About " + slug, Appearances: []domain.Series{{Slug: "series-" + slug, Name: "Series " + slug}}}
}

func rankingOf(n int) *domain.Ranking {
	rows := make([]domain.WaifuSummary, n)
	for i := range rows {
		rows[i] = domain.WaifuSummary{Slug: fmt.Sprintf("ranked-%03d", i), Name: fmt.Sprintf("Ranked %03d", i), Likes: 10000 - i*10, Trash: 5}
	}
	return domain.BuildRanking(rows, domain.DefaultMinVotes, time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC), 1000)
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, loc)}
	rng := &seqRandom{}
	players := memory.NewPlayerStore(clock)
	source := mocks.NewWaifuSource(t)
	source.EXPECT().Get(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, slug string) (domain.Waifu, error) {
		return detail(slug), nil
	}).Maybe()
	ranking := memory.NewRankingHolder(rankingOf(200))
	daily := memory.NewDailyStore()
	wotd := app.NewWotdService(daily, ranking, clock, rng, loc, nil)
	status := &fakeStatus{}
	banners := memory.NewBannerStore()
	banner := app.NewBannerService(banners, ranking, source, clock, rng, loc, app.BannerConfig{}, nil)
	svc := Services{
		Admin:     app.NewAdminService(players, source, clock, nil),
		Roll:      app.NewRollService(players, source, ranking, wotd, banner, clock, rng, 0, nil),
		Daily:     app.NewDailyService(players, clock, rng, loc),
		Coins:     app.NewCoinsService(players),
		Inventory: app.NewInventoryService(players, ranking),
		Search:    app.NewSearchService(source, ranking),
		Trade:     app.NewTradeService(players),
		Wotd:      wotd,
		Banner:    banner,
		Ranking:   ranking,
		Status:    status,
	}
	api := newFakeAPI()
	bot := New(api, svc, Config{AppID: "app", BotUserID: botID, Version: "test", Clock: clock})
	svc.Status = status
	return &fixture{t: t, api: api, bot: bot, clock: clock, rng: rng, players: players, source: source, ranking: ranking, daily: daily, banners: banners, loc: loc, status: status}
}

func (f *fixture) script(vals ...int) { f.rng.vals = append(f.rng.vals, vals...) }

func (f *fixture) seedBanner(slugs ...string) domain.Banner {
	f.t.Helper()
	week := domain.BannerWeekStart(f.clock.now, f.loc)
	series := domain.Series{Slug: "re-zero", Name: "Re:Zero", URL: "https://www.mywaifulist.moe/series/re-zero", PictureURL: "https://img/re-zero", Description: "A boy is summoned."}
	stored, err := f.banners.Put(context.Background(), domain.NewBanner(week, series, rankingOf(200).Subset(slugs)))
	if err != nil {
		f.t.Fatal(err)
	}
	return stored
}

func (f *fixture) fund(userID string, amount int64) {
	f.t.Helper()
	f.give(userID)
	if _, ok, err := f.players.ClaimDaily(context.Background(), domain.PlayerKey{GuildID: guildID, UserID: userID}, amount, f.clock.now.Add(-time.Hour), f.clock.now); err != nil || !ok {
		f.t.Fatalf("setup funding failed: ok=%v err=%v", ok, err)
	}
}

func d100(v int) int { return v - 1 }

func (f *fixture) give(userID string, slugs ...string) {
	f.t.Helper()
	key := domain.PlayerKey{GuildID: guildID, UserID: userID}
	if _, err := f.players.EnsurePlayer(context.Background(), key); err != nil {
		f.t.Fatal(err)
	}
	for i, slug := range slugs {
		w := domain.OwnedFromSummary(summary(slug), f.clock.now.Add(time.Duration(i)*time.Second))
		if _, err := f.players.AddOwned(context.Background(), key, w); err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *fixture) coins(userID string) int64 {
	f.t.Helper()
	p, err := f.players.GetPlayer(context.Background(), domain.PlayerKey{GuildID: guildID, UserID: userID})
	if err != nil {
		f.t.Fatal(err)
	}
	return p.Coins
}

func (f *fixture) owns(userID, slug string) bool {
	got, _ := f.players.OwnedSlugs(context.Background(), domain.PlayerKey{GuildID: guildID, UserID: userID}, []string{slug})
	return len(got) == 1
}

func (f *fixture) nextID() string {
	f.seq++
	return strconv.Itoa(f.seq)
}

func member(userID string) *discordgo.Member {
	return &discordgo.Member{User: &discordgo.User{ID: userID, Username: userID}}
}

func strOpt(name, value string) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{Name: name, Type: discordgo.ApplicationCommandOptionString, Value: value}
}

func boolOpt(name string, value bool) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{Name: name, Type: discordgo.ApplicationCommandOptionBoolean, Value: value}
}

func userOption(userID string) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{Name: optUser, Type: discordgo.ApplicationCommandOptionUser, Value: userID}
}

func resolvedUsers(ids ...string) *discordgo.ApplicationCommandInteractionDataResolved {
	users := map[string]*discordgo.User{}
	for _, id := range ids {
		users[id] = &discordgo.User{ID: id, Username: id}
	}
	return &discordgo.ApplicationCommandInteractionDataResolved{Users: users}
}

func (f *fixture) slash(userID, command, sub string, resolved *discordgo.ApplicationCommandInteractionDataResolved, opts ...*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	data := discordgo.ApplicationCommandInteractionData{Name: command, Resolved: resolved}
	if sub != "" {
		data.Options = []*discordgo.ApplicationCommandInteractionDataOption{{Name: sub, Type: discordgo.ApplicationCommandOptionSubCommand, Options: opts}}
	} else {
		data.Options = opts
	}
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID: f.nextID(), Type: discordgo.InteractionApplicationCommand, GuildID: guildID, ChannelID: channelID, Member: member(userID), Data: data,
	}}
}

func (f *fixture) adminCmd(userID, group, sub string, resolved *discordgo.ApplicationCommandInteractionDataResolved, opts ...*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	ic := f.slash(userID, commandAdmin, sub, resolved, opts...)
	data := ic.Data.(discordgo.ApplicationCommandInteractionData)
	data.Options = []*discordgo.ApplicationCommandInteractionDataOption{{Name: group, Type: discordgo.ApplicationCommandOptionSubCommandGroup, Options: data.Options}}
	ic.Data = data
	ic.Member.Permissions = discordgo.PermissionAdministrator
	return ic
}

func intOpt(name string, value int) *discordgo.ApplicationCommandInteractionDataOption {
	return &discordgo.ApplicationCommandInteractionDataOption{Name: name, Type: discordgo.ApplicationCommandOptionInteger, Value: float64(value)}
}

func (f *fixture) dm(userID, command, sub string, opts ...*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	ic := f.slash(userID, command, sub, nil, opts...)
	ic.GuildID = ""
	ic.Member = nil
	ic.User = &discordgo.User{ID: userID}
	return ic
}

func (f *fixture) click(userID string, msg *discordgo.Message, customID string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID: f.nextID(), Type: discordgo.InteractionMessageComponent, GuildID: guildID, ChannelID: channelID, Member: member(userID), Message: msg,
		Data: discordgo.MessageComponentInteractionData{CustomID: customID},
	}}
}

func (f *fixture) modal(userID string, msg *discordgo.Message, customID, inputID, value string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID: f.nextID(), Type: discordgo.InteractionModalSubmit, GuildID: guildID, ChannelID: channelID, Member: member(userID), Message: msg,
		Data: discordgo.ModalSubmitInteractionData{CustomID: customID, Components: []discordgo.MessageComponent{&discordgo.ActionsRow{Components: []discordgo.MessageComponent{&discordgo.TextInput{CustomID: inputID, Value: value}}}}},
	}}
}

func (f *fixture) autocomplete(userID string, resolved *discordgo.ApplicationCommandInteractionDataResolved, opts ...*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	ic := f.slash(userID, commandWaifu, subTrade, resolved, opts...)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	return ic
}

func (f *fixture) contextMenu(userID string, target *discordgo.Message) *discordgo.InteractionCreate {
	resolved := &discordgo.ApplicationCommandInteractionDataResolved{Messages: map[string]*discordgo.Message{}}
	targetID := ""
	if target != nil {
		targetID = target.ID
		resolved.Messages[target.ID] = target
	}
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID: f.nextID(), Type: discordgo.InteractionApplicationCommand, GuildID: guildID, ChannelID: channelID, Member: member(userID),
		Data: discordgo.ApplicationCommandInteractionData{Name: commandSell, CommandType: discordgo.MessageApplicationCommand, TargetID: targetID, Resolved: resolved},
	}}
}

func (f *fixture) message(id string) *discordgo.Message {
	f.t.Helper()
	f.api.mu.Lock()
	defer f.api.mu.Unlock()
	msg, ok := f.api.messages[id]
	if !ok {
		f.t.Fatalf("message %s not found", id)
	}
	copied := *msg
	return &copied
}

func (f *fixture) run(ic *discordgo.InteractionCreate) {
	f.bot.Handle(ic)
}

func (f *fixture) respondContent() string {
	r := f.api.lastRespond()
	if r == nil || r.Data == nil {
		return ""
	}
	return r.Data.Content
}

func cardName(e *discordgo.MessageEmbed) string {
	parts := strings.Split(e.Title, "\n")
	return parts[len(parts)-1]
}

func hasComponents(cs []discordgo.MessageComponent) bool {
	return len(cs) > 0
}
