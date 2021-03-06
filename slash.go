package bcr

import (
	"strings"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
)

// SyncCommands syncs slash commands in the given guilds.
// If no guilds are given, slash commands are synced globally.
// Router.Bot *must* be set before calling this function or it will panic!
func (r *Router) SyncCommands(guildIDs ...discord.GuildID) (err error) {
	r.cmdMu.Lock()
	cmds := []*Command{}
	for _, cmd := range r.cmds {
		if cmd.Options != nil {
			cmds = append(cmds, cmd)
		}
	}
	r.cmdMu.Unlock()

	slashCmds := []api.CreateCommandData{}
	for _, cmd := range cmds {
		slashCmds = append(slashCmds, api.CreateCommandData{
			Name:        strings.ToLower(cmd.Name),
			Description: cmd.Summary,
			Options:     *cmd.Options,
		})
	}

	if len(guildIDs) > 0 {
		return r.syncCommandsIn(slashCmds, guildIDs)
	}
	return r.syncCommandsGlobal(slashCmds)
}

func (r *Router) syncCommandsGlobal(cmds []api.CreateCommandData) (err error) {
	appID := discord.AppID(r.Bot.ID)
	s, _ := r.StateFromGuildID(0)

	deleted := []discord.CommandID{}
	current, err := s.Commands(appID)
	if err != nil {
		return err
	}

	for _, c := range current {
		if !in(cmds, c.Name) {
			deleted = append(deleted, c.ID)
		}
	}

	for _, id := range deleted {
		err = s.DeleteCommand(appID, id)
		if err != nil {
			return err
		}
	}

	for _, cmd := range cmds {
		_, err = s.CreateCommand(appID, cmd)
		if err != nil {
			return err
		}
	}

	return nil
}

func in(cmds []api.CreateCommandData, name string) bool {
	for _, cmd := range cmds {
		if cmd.Name == name {
			return true
		}
	}
	return false
}

func (r *Router) syncCommandsIn(cmds []api.CreateCommandData, guildIDs []discord.GuildID) (err error) {
	appID := discord.AppID(r.Bot.ID)

	for _, guild := range guildIDs {
		s, _ := r.StateFromGuildID(guild)

		deleted := []discord.CommandID{}
		current, err := s.GuildCommands(appID, guild)
		if err != nil {
			return err
		}

		for _, c := range current {
			if !in(cmds, c.Name) {
				deleted = append(deleted, c.ID)
			}
		}

		for _, id := range deleted {
			err = s.DeleteGuildCommand(appID, guild, id)
			if err != nil {
				return err
			}
		}

		for _, cmd := range cmds {
			_, err = s.CreateGuildCommand(appID, guild, cmd)
			if err != nil {
				return err
			}
		}
	}

	return
}
