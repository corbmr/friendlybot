package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"sync"

	. "github.com/bwmarrin/discordgo"

	// "github.com/golang/groupcache/lru"
	"bytes"
	"io/ioutil"
	"log"
	"regexp"
)

var (
	token string
	kirby []byte
)

func init() {
	flag.StringVar(&token, "t", "", "Bot token")
	flag.Parse()

	if token == "" {
		flag.Usage()
		os.Exit(1)
	}

	data, err := ioutil.ReadFile("kirby.png")
	if err != nil {
		log.Println("Unable to find kirby face")
		return
	}

	kirby = data
}

func main() {
	dg, err := New("Bot " + token)
	if err != nil {
		log.Fatalln("error creating Discord session,", err)
		return
	}

	// dg.AddHandler(readyHandler)
	dg.AddHandler(guildJoinHandler)
	dg.AddHandler(messageCreateHandler)

	err = dg.Open()
	if err != nil {
		log.Fatalln("error opening connection,", err)
		return
	}

	defer dg.Close()

	log.Println("Friendlybot now running")

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, os.Kill)
	<-s

	log.Println("Friendlybot shutting down")
}

type (
	guildID = string
	roleID  = string
	command = string
)

type game struct {
	roleName string
	color    int
}

var commands = map[command]game{
	"melee":    game{"melee-friends", 0xE65F47},
	"smash4":   game{"smash4-friends", 0x42CE96},
	"ultimate": game{"ultimate-friends", 0xf8c869},
}

var roleMap = make(map[guildID]map[command]roleID)
var rw sync.RWMutex

/* func readyHandler(s *Session, r *Ready) {

	for _, g := range r.Guilds {
		log.Println("guild: ", g.Name, g.ID)
		go getOrAddRoles(s, g)
	}

} */

func getOrAddRoles(s *Session, g *Guild) {

	roles, err := s.GuildRoles(g.ID)
	if err != nil {
		log.Println("Unable to get guild roles", err)
		return
	}

	found := make(map[command]roleID)

	// Try to find existing roles
	for _, r := range roles {
	FindGame:
		for c, g := range commands {
			if r.Name == g.roleName {
				found[c] = r.ID
				break FindGame
			}
		}
	}

	// Create any roles that don't exist yet
	for command, game := range commands {
		if _, ok := found[command]; !ok {
			log.Printf("Role %v not found in %v, creating new role\n", game.roleName, g.Name)

			newRole, err := s.GuildRoleCreate(g.ID)
			if err != nil {
				log.Printf("Unable to create role %v for guild %s, %v\n", game.roleName, g.Name, err)
			}

			newRole, err = s.GuildRoleEdit(g.ID, newRole.ID, game.roleName, game.color, false, 0, true)
			if err != nil {
				log.Printf("Unable to edit role %v for guild %s, %v\n", game.roleName, g.Name, err)
			}

			found[command] = newRole.ID
		}
	}

	rw.Lock()
	roleMap[g.ID] = found
	rw.Unlock()
}

func guildJoinHandler(s *Session, g *GuildCreate) {
	log.Println("Joined", g.Name)
	getOrAddRoles(s, g.Guild)
}

var goodBotRegex = regexp.MustCompile(`(?i)^good bot`)

const commandPrefix = "!f"

func messageCreateHandler(s *Session, m *MessageCreate) {

	if goodBotRegex.MatchString(m.Content) && kirby != nil {
		go func() {
			reader := bytes.NewReader(kirby)
			s.ChannelFileSend(m.ChannelID, "kirby.png", reader)
		}()
	}

	fields := strings.Fields(m.Content)

	if len(fields) < 1 || fields[0] != commandPrefix {
		return
	}

	if len(fields) < 2 {
		s.ChannelMessageSend(m.ChannelID, "Usage: !f {melee, smash4, ultimate}")
	}

	if _, ok := commands[fields[1]]; ok {
		toggleRole(s, m.ChannelID, m.Author, fields[1])
	} else {
		s.ChannelMessageSend(m.ChannelID, "Usage: !f {melee, smash4, ultimate}")
	}

}

func toggleRole(s *Session, chID string, u *User, game command) {
	ch, err := s.Channel(chID)
	if err != nil {
		log.Println("Unable to get channel", err)
		return
	}

	m, err := s.GuildMember(ch.GuildID, u.ID)
	if err != nil {
		log.Println("Unable to get member", err)
		return
	}

	rw.RLock()
	rID := roleMap[ch.GuildID][game]
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
		log.Println("Unable to change role")
		return
	}

	s.ChannelMessageSend(chID, "Role changed successfully")

}
