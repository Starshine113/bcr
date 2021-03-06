package bcr

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
)

type reactionInfo struct {
	userID          discord.UserID
	ctx             *Context
	fn              func(*Context)
	deleteOnTrigger bool
	deleteReaction  bool
	respondToRemove bool
}

type reactionKey struct {
	messageID discord.MessageID
	emoji     discord.APIEmoji
}

// ReactionAdd runs when a reaction is added to a message
func (r *Router) ReactionAdd(e *gateway.MessageReactionAddEvent) {
	r.reactionMu.Lock()
	defer r.reactionMu.Unlock()
	if v, ok := r.reactions[reactionKey{
		messageID: e.MessageID,
		emoji:     e.Emoji.APIString(),
	}]; ok {
		// check if the reacting user is the same as the required user
		if v.userID != e.UserID {
			return
		}
		// handle deleting the reaction
		// only delete if:
		// - the user isn't the user the reaction's for
		// - or the reaction is supposed to be deleted
		// - and the user is not the bot user
		if v.deleteReaction && e.GuildID.IsValid() && e.UserID != r.Bot.ID {
			state, _ := r.StateFromGuildID(e.GuildID)

			if p, err := state.Permissions(e.ChannelID, r.Bot.ID); err == nil {
				if p.Has(discord.PermissionManageMessages) {
					state.DeleteUserReaction(e.ChannelID, e.MessageID, e.UserID, e.Emoji.APIString())
				}
			}
		}
		// run the handler
		// fork this off to a goroutine to unlock the reaction mutex immediately
		go v.fn(v.ctx)

		// if the handler should be deleted after running, do that
		if v.deleteOnTrigger {
			delete(r.reactions, reactionKey{
				messageID: e.MessageID,
				emoji:     e.Emoji.APIString(),
			})
		}
	}
}

// ReactionRemove runs when a reaction is removed from a message
func (r *Router) ReactionRemove(ev *gateway.MessageReactionRemoveEvent) {
	r.reactionMu.Lock()
	defer r.reactionMu.Unlock()
	if v, ok := r.reactions[reactionKey{
		messageID: ev.MessageID,
		emoji:     ev.Emoji.APIString(),
	}]; ok {
		if !v.respondToRemove {
			return
		}

		// check if the reacting user is the same as the required user
		if v.userID != ev.UserID {
			return
		}

		// run the handler
		// fork this off to a goroutine to unlock the reaction mutex immediately
		go v.fn(v.ctx)

		// if the handler should be deleted after running, do that
		if v.deleteOnTrigger {
			delete(r.reactions, reactionKey{
				messageID: ev.MessageID,
				emoji:     ev.Emoji.APIString(),
			})
		}
	}
}

// ReactionMessageDelete cleans up old handlers on deleted messages
func (r *Router) ReactionMessageDelete(m *gateway.MessageDeleteEvent) {
	r.DeleteReactions(m.ID)
}

// DeleteReactions deletes all reactions for a message
func (r *Router) DeleteReactions(m discord.MessageID) {
	r.reactionMu.Lock()
	for k := range r.reactions {
		if k.messageID == m {
			delete(r.reactions, k)
		}
	}
	r.reactionMu.Unlock()
}
