package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
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
}

func init() {

	var guild = flag.String("guild", "", "GuildID, if not passed bot registers commands globally")
	var token = flag.String("token", "", "Bot access token")
	var app = flag.String("app", "", "Application ID")
	var removeCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
	flag.Parse()

	if *token == "" {
		log.Fatal("Bot token must not be empty")
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
		log.Fatal("Failed to create discord bot connection")
	}

	discord.AddHandler(interactionHandler)
	discord.AddHandler(func(session *discordgo.Session, ready *discordgo.Ready) {
		log.Printf("Logged in as %s", ready.User.String())
	})

	_, err = discord.ApplicationCommandBulkOverwrite(creds.AppID, creds.GuildID, commands)
	if err != nil {
		log.Fatalf("Could not register commands: %s", err)
	}

	err = discord.Open()
	if err != nil {
		log.Fatalf("Could not open discord session: %s", err)
	}

	// Keyboard interrupter
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = discord.Close()
	if err != nil {
		log.Printf("Couldnt gracefully close discord session: %s", err)
	}

}

func interactionHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	if interaction.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := interaction.ApplicationCommandData()
	if data.Name != "test" {
		return
	}

	testHandler(session, interaction)
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
		log.Fatalf("Error while responding to interaction: %s", err)
	}
}
