package bcr

import (
	"errors"
	"strings"

	"github.com/diamondburned/arikawa/v3/bot/extras/shellwords"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/spf13/pflag"
)

// Errors related to getting the context
var (
	ErrChannel   = errors.New("context: couldn't get channel")
	ErrGuild     = errors.New("context: couldn't get guild")
	ErrNoBotUser = errors.New("context: couldn't get bot user")

	ErrEmptyMessage = errors.New("context: message was empty")
)

// Prefixer returns the prefix used and the length. If the message doesn't start with a valid prefix, it returns -1.
// Note that this function should still use the built-in r.Prefixes for mention prefixes
type Prefixer func(m discord.Message) int

// DefaultPrefixer ...
func (r *Router) DefaultPrefixer(m discord.Message) int {
	for _, p := range r.Prefixes {
		if strings.HasPrefix(strings.ToLower(m.Content), p) {
			return len(p)
		}
	}
	return -1
}

// Context is a command context
type Context struct {
	// Command and Prefix contain the invoked command's name and prefix, respectively.
	// Note that Command won't be accurate if the invoked command was a subcommand, use FullCommandPath for that.
	Command string
	Prefix  string

	FullCommandPath []string

	Args    []string
	RawArgs string

	Flags *pflag.FlagSet

	InternalArgs []string
	pos          int

	State   *state.State
	ShardID int

	Bot *discord.User

	// Info about the message
	Message discord.Message
	Channel *discord.Channel
	Guild   *discord.Guild
	Author  discord.User

	// Note: Member is nil for non-guild messages
	Member *discord.Member

	// The command and the router used
	Cmd    *Command
	Router *Router

	AdditionalParams map[string]interface{}
}

// NewContext returns a new message context
func (r *Router) NewContext(m *gateway.MessageCreateEvent) (ctx *Context, err error) {
	messageContent := m.Content

	var p int
	if p = r.Prefixer(m.Message); p != -1 {
		messageContent = messageContent[p:]
	} else {
		return nil, ErrEmptyMessage
	}
	messageContent = strings.TrimSpace(messageContent)

	message, err := shellwords.Parse(messageContent)
	if err != nil {
		message = strings.Split(messageContent, " ")
	}
	if len(message) == 0 {
		return nil, ErrEmptyMessage
	}
	command := strings.ToLower(message[0])
	args := []string{}
	if len(message) > 1 {
		args = message[1:]
	}

	raw := TrimPrefixesSpace(messageContent, message[0])

	// create the context
	ctx = &Context{
		Command: command,
		Prefix:  m.Content[:p],

		InternalArgs:     args,
		Args:             args,
		Message:          m.Message,
		Author:           m.Author,
		Member:           m.Member,
		RawArgs:          raw,
		Router:           r,
		Bot:              r.Bot,
		AdditionalParams: make(map[string]interface{}),
	}

	ctx.State, ctx.ShardID = r.StateFromGuildID(m.GuildID)

	// get the channel
	ctx.Channel, err = ctx.State.Channel(m.ChannelID)
	if err != nil {
		return ctx, ErrChannel
	}
	// get guild
	if m.GuildID.IsValid() {
		ctx.Guild, err = ctx.State.Guild(m.GuildID)
		if err != nil {
			return ctx, ErrGuild
		}
		ctx.Guild.Roles, err = ctx.State.Roles(m.GuildID)
		if err != nil {
			return ctx, ErrGuild
		}
	}

	return ctx, err
}

// DisplayName returns the context user's displayed name (either username without discriminator, or nickname)
func (ctx *Context) DisplayName() string {
	if ctx.Member == nil {
		return ctx.Author.Username
	}
	if ctx.Member.Nick == "" {
		return ctx.Author.Username
	}
	return ctx.Member.Nick
}
