package main

import (
    "fmt"
    "strings"
    "os"
    "os/signal"
    "flag"
    "time"
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

    dg.AddHandler(messageCreate)

    // Open a websocket connection to Discord and begin listening.
    err = dg.Open()
    if err != nil {
    fmt.Println("error opening connection,", err)
        return
    }

    fmt.Println("Friendlybot now running")

    s := make(chan os.Signal, 1)
    signal.Notify(s, os.Interrupt, os.Kill)
    <-s

    dg.Close()

}

type Game int

const (
	Melee Game = iota
	Smash4
)

var RoleMap := make(map[string][2]string)

func ready(s *Session, r *Ready) {
	for _, r := range r.Guilds {

	}
}

func messageCreate(s *Session, m *MessageCreate) {
	if !strings.HasPrefix(m.Content, "!") {
		return
	}


}