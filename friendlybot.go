package main

import (
    "fmt"
    "strings"
    "os"
    "os/signal"
    "flag"
    "sync"
    . "github.com/bwmarrin/discordgo"
)

var Token string

func init() {
    flag.StringVar(&Token, "t", "", "Bot token")
    flag.Parse()

    if Token == "" {
        flag.Usage()
        os.Exit(1)
    }
}

func main() {
    dg, err := New("Bot " + Token)
    if err != nil {
        fmt.Println("error creating Discord session,", err)
        return
    }

    dg.AddHandler(ReadyHandler)
    dg.AddHandler(GuildJoinHandler)
    dg.AddHandler(MessageCreateHandler)

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

type Game int

const (
	Melee Game = iota
	Smash4
    GameCount
)

var RoleNames = [GameCount]string{
    "Melee Friendlies",
    "Smash 4 Friendlies",
}

type (
    GuildID = string
    RoleID = string
)

var RoleMap sync.Map

func ReadyHandler(s *Session, r *Ready) {

    var wg sync.WaitGroup
	for _, g := range r.Guilds {
        wg.Add(1)
        go func(g *Guild) {
            defer wg.Done()
            GetOrAddRoles(s, g)
        }(g)
	}
    wg.Wait()

}

func GetOrAddRoles(s *Session, g *Guild) {

    var found [GameCount]RoleID

    roles, err := s.GuildRoles(g.ID)
    if err != nil {
        fmt.Println("Unable to get guild roles", err)
        return
    }

    for _, role := range roles {

        switch role.Name {
        case RoleNames[Melee]:
            found[Melee] = role.ID
            fmt.Println("Found existing melee role")

        case RoleNames[Smash4]:
            found[Smash4] = role.ID
            fmt.Println("Found existing smash 4 role")
        }

    }

    if found[Melee] == "" {
        r, err := s.GuildRoleCreate(g.ID)
        if err != nil {
            fmt.Println("Unable to create melee role for guild", g.Name, err)
        }

        r, err = s.GuildRoleEdit(g.ID, r.ID, RoleNames[Melee], 0xFF0000, false, 0, true)
        if err != nil {
            fmt.Println("Unable to change melee role")
        }
    }

    if found[Smash4] == "" {
        r, err := s.GuildRoleCreate(g.ID)
        if err != nil {
            fmt.Println("Unable to create smash4 role for guild", g.Name, err)
        }

        r, err = s.GuildRoleEdit(g.ID, r.ID, RoleNames[Smash4], 0x0000FF, false, 0, true)
        if err != nil {
            fmt.Println("Unable to change smash 4 role")
        }
    }

    RoleMap.Store(g.ID, found)
}

func GuildJoinHandler(s *Session, g *GuildCreate) {
    GetOrAddRoles(s, g.Guild)
}

func MessageCreateHandler(s *Session, m *MessageCreate) {

    command := strings.Fields(m.Content)

    if len(command) < 2 || command[0] != "!f" {
        return
    }

    switch command[1] {
    case "melee":
        ToggleRole(s, m.ChannelID, m.Author, Melee)
    case "smash4":
        ToggleRole(s, m.ChannelID, m.Author, Smash4)

    default:
        s.ChannelMessageSend(m.ChannelID, "Unknown game. Valid options are melee or smash4")
    }

}

func ToggleRole(s *Session, chID string, u *User, game Game) {
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

    rs, ok := RoleMap.Load(ch.GuildID)
    if !ok {
        fmt.Println("Unable to load role", err)
    }

    rID := rs.([GameCount]RoleID)[game]

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
