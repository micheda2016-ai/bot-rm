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
	// Server per Render
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "Bot Online") })
		port := os.Getenv("PORT")
		if port == "" { port = "8080" }
		http.ListenAndServe(":"+port, nil)
	}()

	token := os.Getenv("DISCORD_TOKEN")
	s, err := discordgo.New("Bot " + token)
	if err != nil { log.Fatalf("Errore: %v", err) }

	commands := []*discordgo.ApplicationCommand{
		{Name: "setup-ticket", Description: "Configura il pannello ticket"},
		{Name: "chiama-fdo", Description: "Invia notifica alla Categoria FDO"},
		{
			Name: "promozione",
			Description: "Annuncia una promozione",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "Utente da promuovere", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "grado", Description: "Scrivi il nuovo grado", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivazione", Required: true},
			},
		},
		{
			Name: "retrocessione",
			Description: "Annuncia una retrocessione",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "Utente da retrocedere", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "grado", Description: "Scrivi il grado assegnato", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivazione", Required: true},
			},
		},
		// --- COMANDO ARRESTO AGGIORNATO ---
		{
			Name: "arresto",
			Description: "Registra un arresto nel sistema",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "discord-civile", Description: "Tag Discord del civile arrestato", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "roblox-civile", Description: "Nome Roblox del civile arrestato", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "roblox-agente", Description: "Tuo nome Roblox (Agente)", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivo dell'arresto", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "verbale", Description: "Codice o link del verbale", Required: true},
			},
		},
		// --- COMANDO MULTA ---
		{
			Name: "multa",
			Description: "Registra una multa nel sistema",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "discord-civile", Description: "Tag Discord del civile", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "roblox-civile", Description: "Nome Roblox del civile", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "roblox-agente", Description: "Tuo nome Roblox (Agente)", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "valore", Description: "Importo (1000-8000)", Required: true, MinValue: &[]float64{1000}[0], MaxValue: 8000},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivo della multa", Required: true},
			},
		},
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			data := i.ApplicationCommandData()
			switch data.Name {
			case "arresto":
				uDiscord := data.Options[0].UserValue(s)
				res := fmt.Sprintf("⚖️ **REGISTRO ARRESTI UFFICIALE**\n\n"+
					"👤 **Civile (Discord):** %s\n"+
					"🆔 **Civile (Roblox):** %s\n"+
					"👮 **Agente (Roblox):** %s\n"+
					"📝 **Motivo:** %s\n"+
					"📂 **Verbale:** %s", 
					uDiscord.Mention(), data.Options[1].StringValue(), data.Options[2].StringValue(), data.Options[3].StringValue(), data.Options[4].StringValue())
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: 4, Data: &discordgo.InteractionResponseData{Content: res}})

			case "multa":
				uDiscord := data.Options[0].UserValue(s)
				res := fmt.Sprintf("🧾 **VERBALE DI MULTA**\n\n"+
					"👤 **Civile (Discord):** %s\n"+
					"🆔 **Civile (Roblox):** %s\n"+
					"👮 **Agente (Roblox):** %s\n"+
					"💰 **Valore:** %d$\n"+
					"📝 **Motivo:** %s", 
					uDiscord.Mention(), data.Options[1].StringValue(), data.Options[2].StringValue(), data.Options[3].IntValue(), data.Options[4].StringValue())
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: 4, Data: &discordgo.InteractionResponseData{Content: res}})

			case "promozione":
				u := data.Options[0].UserValue(s)
				res := fmt.Sprintf("🎖️ **PROMOZIONE UFFICIALE**\n\n**Soggetto:** %s\n**Nuovo Grado:** %s\n**Motivazione:** %s", u.Mention(), data.Options[1].StringValue(), data.Options[2].StringValue())
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: 4, Data: &discordgo.InteractionResponseData{Content: res}})

			case "chiama-fdo":
				ruoloFDO := "1492918778885963836"
				mappa := "ijisma95"
				msg := fmt.Sprintf("<@&%s>\n🚨 **CHIAMATA FORZE DELL'ORDINE** 🚨\n\n👤 **Mittente:** <@%s>\n📍 **Cod Mappa EH:** `%s`\n⚠️ Intervento richiesto immediatamente!", ruoloFDO, i.Member.User.ID, mappa)
				s.ChannelMessageSend(i.ChannelID, msg)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: 4, Data: &discordgo.InteractionResponseData{Content: "✅ Chiamata inviata!", Flags: 64}})
            
            // ... (altri comandi come ticket rimangono uguali)
			}
		}
	})

	s.Open()
	s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", commands)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
