package main

import (
	"fmt"
	. "github.com/bwmarrin/discordgo"
	"os"
	"os/signal"
	"strings"
	"sync"
)

var Token string

func init() {
	Token = os.Getenv("$FRIENDTOKEN")

	if Token == "" {
		fmt.Println("FRIENDTOKEN environment variable not set")
		os.Exit(1)
	}
}

func main() {
	dg, err := New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(readyHandler)
	dg.AddHandler(guildJoinHandler)
	dg.AddHandler(messageCreateHandler)

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	defer dg.Close()

	fmt.Println("Friendlybot now running")

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, os.Kill)
	<-s
}

type game struct {
	roleName, command string
	color             int
}

var games = [...]game{
	game{"melee-friends", "melee", 0xE65F47},
	game{"smash4-friends", "smash4", 0x479CF1},
}

type (
	guildID = string
	roleID  = string
)

var roleMap = make(map[guildID][]roleID)
var rw sync.RWMutex

func readyHandler(s *Session, r *Ready) {

	var wg sync.WaitGroup
	for _, g := range r.Guilds {
		wg.Add(1)
		go func(g *Guild) {
			defer wg.Done()
			getOrAddRoles(s, g)
		}(g)
	}
	wg.Wait()

}

func getOrAddRoles(s *Session, g *Guild) {

	roles, err := s.GuildRoles(g.ID)
	if err != nil {
		fmt.Println("Unable to get guild roles", err)
		return
	}

	found := make([]roleID, len(games))

	for _, r := range roles {
		for i, g := range games {
			if r.Name == g.roleName {
				found[i] = r.ID
				break
			}
		}
	}

	for i := range found {
		if found[i] == "" {
			r, err := s.GuildRoleCreate(g.ID)
			if err != nil {
				fmt.Printf("Unable to create role %v for guild %s, %v\n", games[i].roleName, g.Name, err)
			}

			r, err = s.GuildRoleEdit(g.ID, r.ID, games[i].roleName, games[i].color, false, 0, true)
			if err != nil {
				fmt.Printf("Unable to edit role %v for guild %s, %v\n", games[i].roleName, g.Name, err)
			}

			found[i] = r.ID
		}
	}

	rw.Lock()
	roleMap[g.ID] = found
	rw.Unlock()
}

func guildJoinHandler(s *Session, g *GuildCreate) {
	getOrAddRoles(s, g.Guild)
}

func messageCreateHandler(s *Session, m *MessageCreate) {

	command := strings.Fields(m.Content)

	if len(command) < 2 || command[0] != "!f" {
		return
	}

	if command[1] == "list" {
		s.ChannelMessageSend(m.ChannelID, "Supported games are melee and smash4")
		return
	}

	for i, g := range games {
		if command[1] == g.command {
			toggleRole(s, m.ChannelID, m.Author, i)
			return
		}
	}

	s.ChannelMessageSend(m.ChannelID, "Unknown game.")

}

func toggleRole(s *Session, chID string, u *User, g int) {
	ch, err := s.Channel(chID)
	if err != nil {
		fmt.Println("Unable to get channel", err)
		return
	}

	m, err := s.GuildMember(ch.GuildID, u.ID)
	if err != nil {
		fmt.Println("Unable to get member", err)
		return
	}

    rw.RLock()
	rID := roleMap[ch.GuildID][g]
    rw.RUnlock()

	has := false
	for _, r := range m.Roles {
		if r == rID {
			has = true
			break
		}
	}

	if has {
		err = s.GuildMemberRoleRemove(ch.GuildID, u.ID, rID)
	} else {
		err = s.GuildMemberRoleAdd(ch.GuildID, u.ID, rID)
	}

	if err != nil {
		fmt.Println("Unable to change role")
		return
	}

	s.ChannelMessageSend(chID, "Role changed successfully")

}
