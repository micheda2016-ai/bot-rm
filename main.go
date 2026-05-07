package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
)

func main() {
	// Protezione per Render (evita lo spegnimento per timeout)
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "Bot Online!") })
		port := os.Getenv("PORT")
		if port == "" { port = "8080" }
		http.ListenAndServe(":"+port, nil)
	}()

	token := os.Getenv("DISCORD_TOKEN")
	s, err := discordgo.New("Bot " + token)
	if err != nil { log.Fatalf("Errore sessione: %v", err) }

	// DEFINIZIONE COMANDI CON OPZIONI
	commands := []*discordgo.ApplicationCommand{
		{Name: "setup-ticket", Description: "Crea il pannello per aprire i ticket"},
		{Name: "chiama-fdo", Description: "Invia una notifica urgente alla Categoria FDO"},
		{
			Name: "promozione",
			Description: "Comunica una promozione ufficiale",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "L'utente da promuovere", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "nuovo-ruolo", Description: "Il grado ottenuto", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivo della promozione", Required: true},
			},
		},
		{
			Name: "retrocessione",
			Description: "Comunica una retrocessione",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "L'utente da retrocedere", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "vecchio-ruolo", Description: "Il grado perso", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivo del provvedimento", Required: true},
			},
		},
		{
			Name: "avvertimento",
			Description: "Invia un avvertimento ufficiale",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "L'utente da avvertire", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivo dell'avvertimento", Required: true},
			},
		},
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			data := i.ApplicationCommandData()
			
			switch data.Name {
			case "chiama-fdo":
				// Ping specifico per la Categoria FDO usando l'ID fornito
				ping := "<@&1492918778885963836>"
				res := fmt.Sprintf("🚨 **CHIAMATA DI EMERGENZA** 🚨\n\n**Destinatari:** %s\n**Messaggio:** È richiesto l'intervento immediato di una pattuglia!", ping)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: res},
				})

			case "promozione":
				target := data.Options[0].UserValue(s)
				res := fmt.Sprintf("🎖️ **PROMOZIONE UFFICIALE**\n\n**Soggetto:** %s\n**Nuovo Grado:** %s\n**Motivazione:** %s", target.Mention(), data.Options[1].StringValue(), data.Options[2].StringValue())
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: res},
				})

			case "retrocessione":
				target := data.Options[0].UserValue(s)
				res := fmt.Sprintf("📉 **RETROCESSIONE DI GRADO**\n\n**Soggetto:** %s\n**Grado Rimosso:** %s\n**Motivazione:** %s", target.Mention(), data.Options[1].StringValue(), data.Options[2].StringValue())
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: res},
				})

			case "avvertimento":
				target := data.Options[0].UserValue(s)
				res := fmt.Sprintf("⚠️ **AVVERTIMENTO UFFICIALE**\n\n**Soggetto:** %s\n**Motivazione:** %s\n\n*Si prega di seguire il regolamento per evitare ulteriori provvedimenti.*", target.Mention(), data.Options[1].StringValue())
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: res},
				})

			case "setup-ticket":
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "📩 **CENTRO ASSISTENZA**\nClicca il tasto qui sotto per parlare con lo Staff in un canale privato.",
						Components: []discordgo.MessageComponent{
							discordgo.ActionsRow{Components: []discordgo.MessageComponent{
								discordgo.Button{Label: "Apri Ticket", Style: discordgo.PrimaryButton, CustomID: "open_ticket", Emoji: discordgo.ComponentEmoji{Name: "📩"}},
							}},
						},
					},
				})
			}
		}
		
		// Gestione click bottone Ticket
		if i.Type == discordgo.InteractionMessageComponent && i.MessageComponentData().CustomID == "open_ticket" {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "✅ **Richiesta Inviata!** Un amministratore è stato notificato e ti assisterà a breve.",
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			// Ping allo staff nel canale quando un ticket viene aperto
			s.ChannelMessageSend(i.ChannelID, "🔔 **NOTIFICA STAFF:** L'utente "+i.Member.User.Mention()+" ha aperto un ticket!")
		}
	})

	s.Open()
	for _, v := range commands { s.ApplicationCommandCreate(s.State.User.ID, "", v) }
	fmt.Println("Bot pronto con ID FDO salvato!")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
