package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"time"
)

type DiscordCredentials struct {
	GuildID        string
	BotToken       string
	AppID          string
	RemoveCommands bool
}

var creds DiscordCredentials

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "test",
		Description: "This is a test command!",
	},
	{
		Name:        "join",
		Description: "Join your active Voice Channel",
	},
	{
		Name:        "quote",
		Description: "Random quote!",
	},
}

func init() {

	var guild = flag.String("guild", "", "GuildID, if not passed bot registers commands globally")
	var token = flag.String("token", "", "Bot access token")
	var app = flag.String("app", "", "Application ID")
	var removeCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
	flag.Parse()

	if *token == "" {
		log.Print("Bot token must not be empty")
	}

	creds = DiscordCredentials{
		*guild,
		*token,
		*app,
		*removeCommands,
	}

}

func main() {

	discord, err := discordgo.New("Bot " + creds.BotToken)
	if err != nil {
		log.Print("Failed to create discord bot connection")
	}

	discord.AddHandler(interactionHandler)
	discord.AddHandler(func(session *discordgo.Session, ready *discordgo.Ready) {
		log.Printf("Logged in as %s\n", ready.User.String())
	})

	_, err = discord.ApplicationCommandBulkOverwrite(creds.AppID, creds.GuildID, commands)

	if err != nil {
		log.Printf("Could not register commands: %s\n", err)
	}

	err = discord.Open()
	if err != nil {
		log.Printf("Could not open discord session: %s\n", err)
	}

	// Keyboard interrupter
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = discord.Close()
	if err != nil {
		log.Printf("Couldnt gracefully close discord session: %s\n", err)
	}

}

func interactionHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	if interaction.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := interaction.ApplicationCommandData()
	if data.Name == "test" {
		testHandler(session, interaction)
	}
	if data.Name == "join" {
		joinHandler(session, interaction)
	}
	if data.Name == "quote" {
		quoteHandler(session, interaction)
	}

	return

}

func joinHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) (err error) {

	// Sending text response
	returnString := "Attempting to join users Voice Channel..."
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: returnString,
		},
	}
	err = session.InteractionRespond(interaction.Interaction, response)
	if err != nil {
		log.Printf("Error while responding to interaction: %s\n", err)
	}

	// Find what VC to join
	var guildID string
	var channelID string

	// Find the channel the message came from
	channel, err := session.State.Channel(interaction.ChannelID)
	if err != nil {
		log.Printf("Couldnt find channel for join command: %s\n", err)
	}

	// Find Guild channel belongs to
	guild, err := session.State.Guild(channel.GuildID)
	if err != nil {
		log.Printf("Couldnt find matching guild for channel: %s\n", err)
	}

	//	Look at VCs and find user
	for _, voiceStatus := range guild.VoiceStates {
		voiceID := voiceStatus.UserID
		interactionID := interaction.Member.User.ID
		if voiceID == interactionID {
			fmt.Printf("Found user %s in channel %s\n", interaction.Member.User.Username, voiceStatus.ChannelID)
			channelID = voiceStatus.ChannelID
		}
	}

	if channelID == "" {
		log.Println("Couldnt find user who ran join")
	}

	// Join the voice channel
	log.Printf("Attempting to join voice channel\n")
	// guildID = "790405217803305000"
	// channelID = "885998792640458783"
	voiceSession, err := session.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		log.Printf("Error while attempting to join server: %s\n", err)
	}

	voiceSession.Speaking(true)

	time.Sleep(10 * time.Second)
	voiceSession.Disconnect()

	return nil

}

func testHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	returnString := "Wow lookie here it worked!"

	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: returnString,
		},
	}

	err := session.InteractionRespond(interaction.Interaction, response)
	if err != nil {
		log.Printf("Error while responding to interaction: %s", err)
	}
}

func quoteHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	quotes := []string{"test1", "test2", "test3"}

	returnString := quotes[rand.IntN(len(quotes))]

	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: returnString,
		},
	}

	err := session.InteractionRespond(interaction.Interaction, response)
	if err != nil {
		log.Printf("Error while responding to interaction: %s", err)
	}

}
