package main

import (
	"context"
	"github.com/joho/godotenv"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/voice"

	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/ffmpeg-audio"
	"github.com/disgoorg/snowflake/v2"
)

type DiscordCredentials struct {
	GuildID  snowflake.ID
	BotToken string
}

var creds DiscordCredentials

var commands = []discord.ApplicationCommandCreate{
	discord.SlashCommandCreate{
		Name:        "test",
		Description: "This is a test command!",
	},
	discord.SlashCommandCreate{
		Name:        "join",
		Description: "Join your active Voice Channel",
	},
	discord.SlashCommandCreate{
		Name:        "leave",
		Description: "Leave the current voice channel",
	},
	discord.SlashCommandCreate{
		Name:        "quote",
		Description: "Random quote!",
	},
	discord.SlashCommandCreate{
		Name:        "echo",
		Description: "says what you say",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "message",
				Description: "What to say",
				Required:    true,
			},
			discord.ApplicationCommandOptionBool{
				Name:        "ephemeral",
				Description: "If the response should only be visible to you",
				Required:    true,
			},
		},
	},
}

type AudioStream struct {
	ChannelID  *snowflake.ID
	Connection *voice.Conn
}

// Global list of joined channels for leaving
var audioStreams []AudioStream

// kill channel
var sigch = make(chan os.Signal, 1)

func init() {

	// Load dotenv
	err := godotenv.Load()
	if err != nil {
		slog.Error("Failed to read dotenv", slog.Any("err", err))
	}

	guild_string := os.Getenv("DISCORD_GUILD_ID")
	token := os.Getenv("DISCORD_TOKEN")

	guild, err := snowflake.Parse(guild_string)
	if err != nil {
		slog.Error("Error parsing guild ID", slog.Any("err", err))
	}

	if token == "" {
		slog.Error("Bot token must not be empty")
	}

	creds = DiscordCredentials{
		guild,
		token,
	}

}

func main() {

	discord, err := disgo.New(creds.BotToken,
		bot.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentGuildVoiceStates)),
		bot.WithEventListenerFunc(interactionHandler),
	)
	if err != nil {
		slog.Error("Failed to create discord bot connection", slog.Any("err", err))
		return
	}

	defer discord.Close(context.TODO())

	// Set Commands
	_, err = discord.Rest().SetGuildCommands(discord.ApplicationID(), creds.GuildID, commands)

	err = discord.OpenGateway(context.TODO())
	if err != nil {
		slog.Error("Error while connecting to discord gateway", slog.Any("err", err))
		return
	}

	slog.Info("Bot is now running, press CTL+C to exit")

	// Keyboard interrupter
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sigch

}

func interactionHandler(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	if data.CommandName() == "test" {
		slog.Info("Got test command")
		testHandler(event)
	} else if data.CommandName() == "echo" {
		slog.Info("Got Info command")
		echoHandler(event, &data)
	} else if data.CommandName() == "join" {
		slog.Info("Got join command")
		go joinHandler(event.Client(), event)
	} else if data.CommandName() == "leave" {
		slog.Info("Got leave command")
		go leaveHandler(event.Client(), event)
	}
}

func testHandler(event *events.ApplicationCommandInteractionCreate) {
	const returnString = "Wow lookie here it worked!"

	err := event.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent(returnString).
		SetEphemeral(false).
		Build(),
	)
	if err != nil {
		slog.Error("Error sending response", slog.Any("err", err))
	}

}
func echoHandler(event *events.ApplicationCommandInteractionCreate, data *discord.SlashCommandInteractionData) {

	err := event.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent(data.String("message")).
		SetEphemeral(data.Bool("ephemeral")).
		Build(),
	)
	if err != nil {
		slog.Error("Error sending response", slog.Any("err", err))
	}

}

func leaveHandler(client bot.Client, event *events.ApplicationCommandInteractionCreate) {

	// Find user
	userId := event.User().ID
	slog.Info("Finding channel ID for userID: ", slog.Any("snowflake.ID", userId))
	voiceState, err := client.Rest().GetUserVoiceState(creds.GuildID, userId)
	if err != nil {
		slog.Error("Failed to get voice status for user")
		// Send failed message
		err := event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent("Failed to find user voice channel. Are you in a voice channel?").
			SetEphemeral(false).
			Build(),
		)
		if err != nil {
			slog.Error("Error sending response", slog.Any("err", err))
		}
		return
	}
	slog.Info("Got snowflake channel id for user", slog.Any("snowflake.ID", *voiceState.ChannelID))

	const leaveMessage = "Leaving voice channel!"

	// Send connecting message
	err = event.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent(leaveMessage).
		SetEphemeral(false).
		Build(),
	)
	if err != nil {
		slog.Error("Error sending response", slog.Any("err", err))
	}

	// Leave voice
	if len(audioStreams) < 1 {
		slog.Info("No Audio streams in audiostream, skipping leave command")
		return
	}
	for index, stream := range audioStreams {
		slog.Info("Dropping stream from list: ", slog.Any("AudioStream", stream))
		if *stream.ChannelID == *voiceState.ChannelID {
			// Leave channel
			slog.Info("Found applicable audio stream, leaving", slog.Any("ChannelID", *stream.ChannelID))
			conn := *stream.Connection
			conn.Close(context.TODO())

			// Drop from list
			audioStreams = append(audioStreams[:index], audioStreams[index+1:]...)
		}

	}

}

func joinHandler(client bot.Client, event *events.ApplicationCommandInteractionCreate) {

	// Find user
	userId := event.User().ID
	slog.Info("Finding channel ID for userID: ", slog.Any("snowflake.ID", userId))
	voiceState, err := client.Rest().GetUserVoiceState(creds.GuildID, userId)
	if err != nil {
		slog.Error("Failed to get voice status for user")
		// Send failed message
		err := event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent("Failed to find user voice channel. Are you in a voice channel?").
			SetEphemeral(false).
			Build(),
		)
		if err != nil {
			slog.Error("Error sending response", slog.Any("err", err))
		}
		return
	}
	slog.Info("Got snowflake channel id for user", slog.Any("snowflake.ID", *voiceState.ChannelID))

	const joinMessage = "Attempting to join specified voice channel!"

	// Send connecting message
	err = event.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent(joinMessage).
		SetEphemeral(false).
		Build(),
	)
	if err != nil {
		slog.Error("Error sending response", slog.Any("err", err))
	}

	// Connect to voice
	conn := client.VoiceManager().CreateConn(voiceState.GuildID)
	audioStreams = append(audioStreams, AudioStream{voiceState.ChannelID, &conn})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err = conn.Open(ctx, *voiceState.ChannelID, false, false)
	if err != nil {
		slog.Error("Error connecting to voice channel", slog.Any("err", err))
	}

	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), time.Second*10)
		defer closeCancel()
		conn.Close(closeCtx)

		for index, stream := range audioStreams {
			slog.Info("Dropping stream from list: ", slog.Any("AudioStream", stream))
			if *stream.ChannelID == *voiceState.ChannelID {
				audioStreams = append(audioStreams[:index], audioStreams[index+1:]...)
			}
		}

	}()

	err = conn.SetSpeaking(ctx, voice.SpeakingFlagMicrophone)
	if err != nil {
		panic("error setting speaking flag: " + err.Error())
	}

	writeOpus(conn)

}

func writeOpus(connection voice.Conn) {

	read, write := io.Pipe()
	radioChannel := os.Getenv("RADIO_STREAM_URL")

	go func() {
		defer write.Close()
		response, err := http.Get(radioChannel)
		if err != nil {
			return
		}
		defer response.Body.Close()
		io.Copy(write, response.Body)
	}()

	opusProvider, err := ffmpeg.New(context.Background(), read)
	if err != nil {
		slog.Error("Failed to create opus provider", slog.Any("err", err))
	}

	defer opusProvider.Close()

	connection.SetOpusFrameProvider(opusProvider)
	err = opusProvider.Wait()
	if err != nil {
		slog.Error("Error waiting for opus provider", slog.Any("err", err))
	}

}
