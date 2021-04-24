package bcr

import (
	"errors"
	"fmt"
	"math"

	"github.com/diamondburned/arikawa/v2/discord"
)

// ErrNoEmbeds is returned if PagedEmbed() is called without any embeds
var ErrNoEmbeds = errors.New("PagedEmbed: no embeds")

// PagedEmbed sends a slice of embeds, and attaches reaction handlers to flip through them.
// if extendedReactions is true, also add delete, first page, and last page reactions.
func (ctx *Context) PagedEmbed(embeds []discord.Embed, extendedReactions bool) (msg *discord.Message, err error) {
	// if there's no embeds, return
	if len(embeds) == 0 {
		return nil, ErrNoEmbeds
	}

	// set additional parameters, used for pagination
	ctx.AdditionalParams["page"] = 0
	ctx.AdditionalParams["embeds"] = embeds

	// send the first embed
	msg, err = ctx.Send("", &embeds[0])
	if err != nil {
		return
	}

	// add :x: handler
	ctx.AddReactionHandler(msg.ID, ctx.Author.ID, "❌", true, false, func(*Context) {
		err = ctx.State.DeleteMessage(ctx.Channel.ID, msg.ID)
		if err != nil {
			ctx.Router.Logger.Error("deleting message: %v", err)
		}
	})

	// if there's only one embed, that's it! no pager emoji needed
	if len(embeds) == 1 {
		return
	}

	// react with all required emoji--afawk there's no more concise way to do this
	if extendedReactions {
		if err = ctx.State.React(ctx.Channel.ID, msg.ID, "❌"); err != nil {
			return
		}
		if err = ctx.State.React(ctx.Channel.ID, msg.ID, "⏪"); err != nil {
			return
		}
	}
	if err = ctx.State.React(ctx.Channel.ID, msg.ID, "⬅️"); err != nil {
		return
	}
	if err = ctx.State.React(ctx.Channel.ID, msg.ID, "➡️"); err != nil {
		return
	}
	if extendedReactions {
		if err = ctx.State.React(ctx.Channel.ID, msg.ID, "⏩"); err != nil {
			return
		}
	}

	// add handlers for the reactions
	ctx.AddReactionHandler(msg.ID, ctx.Author.ID, "⬅️", false, true, func(ctx *Context) {
		page := ctx.AdditionalParams["page"].(int)
		embeds := ctx.AdditionalParams["embeds"].([]discord.Embed)

		if page == 0 {
			ctx.State.EditEmbed(ctx.Channel.ID, msg.ID, embeds[len(embeds)-1])
			ctx.AdditionalParams["page"] = len(embeds) - 1
			return
		}
		ctx.State.EditEmbed(ctx.Channel.ID, msg.ID, embeds[page-1])
		ctx.AdditionalParams["page"] = page - 1
	})

	ctx.AddReactionHandler(msg.ID, ctx.Author.ID, "➡️", false, true, func(ctx *Context) {
		page := ctx.AdditionalParams["page"].(int)
		embeds := ctx.AdditionalParams["embeds"].([]discord.Embed)

		if page >= len(embeds)-1 {
			ctx.State.EditEmbed(ctx.Channel.ID, msg.ID, embeds[0])
			ctx.AdditionalParams["page"] = 0
			return
		}
		ctx.State.EditEmbed(ctx.Channel.ID, msg.ID, embeds[page+1])
		ctx.AdditionalParams["page"] = page + 1
	})

	if extendedReactions {
		ctx.AddReactionHandler(msg.ID, ctx.Author.ID, "⏪", false, true, func(ctx *Context) {
			embeds := ctx.AdditionalParams["embeds"].([]discord.Embed)

			ctx.State.EditEmbed(ctx.Channel.ID, msg.ID, embeds[0])
			ctx.AdditionalParams["page"] = 0
		})

		ctx.AddReactionHandler(msg.ID, ctx.Author.ID, "⏩", false, true, func(ctx *Context) {
			embeds := ctx.AdditionalParams["embeds"].([]discord.Embed)

			ctx.State.EditEmbed(ctx.Channel.ID, msg.ID, embeds[len(embeds)-1])
			ctx.AdditionalParams["page"] = len(embeds) - 1
		})
	}
	return
}

// FieldPaginator paginates embed fields, for use in ctx.PagedEmbed
func FieldPaginator(title, description string, colour discord.Color, fields []discord.EmbedField, perPage int) []discord.Embed {
	var (
		embeds []discord.Embed
		count  int

		pages = 1
		buf   = discord.Embed{
			Title:       title,
			Description: description,
			Color:       colour,
			Footer: &discord.EmbedFooter{
				Text: fmt.Sprintf("Page 1/%v", math.Ceil(float64(len(fields))/float64(perPage))),
			},
		}
	)

	for _, field := range fields {
		if count >= perPage {
			embeds = append(embeds, buf)
			buf = discord.Embed{
				Title:       title,
				Description: description,
				Color:       colour,
				Footer: &discord.EmbedFooter{
					Text: fmt.Sprintf("Page %v/%v", pages+1, math.Ceil(float64(len(fields))/float64(perPage))),
				},
			}
			count = 0
			pages++
		}
		buf.Fields = append(buf.Fields, field)
		count++
	}

	if len(buf.Fields) > 0 {
		embeds = append(embeds, buf)
	}

	return embeds
}
