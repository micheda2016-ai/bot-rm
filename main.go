package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	// Recupera il token dalle Environment Variables di Render
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("ERRORE: La variabile d'ambiente DISCORD_TOKEN non è impostata su Render!")
	}

	// Creazione sessione Discord
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Errore durante la creazione della sessione Discord: %v", err)
	}

	// Registrazione gestore degli eventi (Interazioni con Bottoni, Menu e Modals)
	dg.AddHandler(interactionHandler)

	// Apertura connessione con Discord
	err = dg.Open()
	if err != nil {
		log.Fatalf("Errore durante l'apertura della connessione: %v", err)
	}
	defer dg.Close()

	log.Println("✅ Bot 'Rome Italy 2.0' avviato con successo su Render!")

	// Registrazione Comandi Slash
	registerCommands(dg)

	// Mantiene il bot in esecuzione
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Arresto del bot...")
}

// Registrazione comandi Slash
func registerCommands(s *discordgo.Session) {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "setup",
			Description: "Configura i pannelli interattivi del server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "server",
					Description: "Pannello di configurazione generale del bot",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "apri-conto",
					Description: "Invia il pannello bancario per l'apertura del conto",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "bonifico",
					Description: "Invia il pannello per l'invio dei bonifici",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "busta-paga",
					Description: "Invia il pannello per la riscossione dello stipendio",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "cittadinanza",
					Description: "Invia il pannello anagrafe per la carta d'identità",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "porto-darmi",
					Description: "Invia il pannello per la richiesta del porto d'armi",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "immatricolazione",
					Description: "Invia il pannello PRA per immatricolare i veicoli",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "registra-arresto",
					Description: "Invia il pannello delle Forze dell'Ordine per registrare arresti",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "emetti-multa",
					Description: "Invia il pannello per emettere sanzioni stradali",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "staff-panel",
					Description: "Invia il pannello riservato alla moderazione Staff",
				},
			},
		},
	}

	for _, cmd := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			log.Printf("Errore nella registrazione del comando '%s': %v", cmd.Name, err)
		}
	}
}

// Gestore centrale di tutte le interazioni (Comandi, Bottoni, Select Menus, Modals)
func interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleSlashCommands(s, i)
	case discordgo.InteractionMessageComponent:
		handleComponents(s, i)
	case discordgo.InteractionModalSubmit:
		handleModalSubmit(s, i)
	}
}

// Gestione Esecuzione Comandi Slash
func handleSlashCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		return
	}

	subCmd := options[0].Name

	switch subCmd {
	case "server":
		sendServerSetupPanel(s, i)
	case "apri-conto":
		sendApriContoPanel(s, i)
	case "busta-paga":
		sendBustaPagaPanel(s, i)
	case "cittadinanza":
		sendCittadinanzaPanel(s, i)
	case "registra-arresto":
		sendRegistraArrestoPanel(s, i)
	default:
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ Pannello `/setup %s` inviato con successo nel canale!", subCmd),
			},
		})
	}
}

// ---------------------------------------------------------
// PANNELLI EMBED CON BOTTONI E MENU A TENDINA
// ---------------------------------------------------------

// Pannello /setup server
func sendServerSetupPanel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "⚙️ PANNELLO DI CONFIGURAZIONE GENERALE - ROME ITALY 2.0",
		Description: "Seleziona dal menu a tendina sottostante la sezione che desideri configurare:",
		Color:       0x0099FF,
	}

	selectMenu := discordgo.SelectMenu{
		CustomID:    "setup_server_select",
		Placeholder: "Scegli la categoria da configurare...",
		Options: []discordgo.SelectMenuOption{
			{
				Label:       "Economia & Banche",
				Value:       "cfg_economy",
				Description: "Configura stipendi per ruolo, canali banche e cooldown",
				Emoji:       discordgo.ComponentEmoji{Name: "💰"},
			},
			{
				Label:       "Anagrafe & Motorizzazione",
				Value:       "cfg_anagrafe",
				Description: "Imposta ruoli cittadini, canali PRA e carte d'identità",
				Emoji:       discordgo.ComponentEmoji{Name: "🇮🇹"},
			},
			{
				Label:       "Forze dell'Ordine & Giustizia",
				Value:       "cfg_police",
				Description: "Gestisci ruoli abilitati ad arresti e registro verbali",
				Emoji:       discordgo.ComponentEmoji{Name: "🚨"},
			},
			{
				Label:       "Staff & Moderazione",
				Value:       "cfg_staff",
				Description: "Configura permessi Warn, Ban RP e canali log",
				Emoji:       discordgo.ComponentEmoji{Name: "🛡️"},
			},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{selectMenu}},
			},
		},
	})
}

// Pannello /setup apri-conto
func sendApriContoPanel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "🏛️ APERTURA CONTO CORRENTE - ROME ITALY 2.0",
		Description: "Clicca sul bottone sottostante per scegliere il tuo istituto bancario e registrare il tuo conto corrente.",
		Color:       0x2ECC71,
	}

	button := discordgo.Button{
		Label:    "Apri Conto Corrente",
		Style:    discordgo.SuccessButton,
		CustomID: "btn_apri_conto",
		Emoji:    discordgo.ComponentEmoji{Name: "💳"},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{button}},
			},
		},
	})
}

// Pannello /setup busta-paga
func sendBustaPagaPanel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "💼 RISCOSSIONE BUSTA PAGA - ROME ITALY 2.0",
		Description: "Clicca sul bottone per riscattare lo stipendio settimanale. Il bot riconoscerà automaticamente il tuo ruolo fazione e accrediterà l'importo sul tuo conto.",
		Color:       0xF1C40F,
	}

	button := discordgo.Button{
		Label:    "Riscuoti Busta Paga",
		Style:    discordgo.PrimaryButton,
		CustomID: "btn_riscuoti_stipendio",
		Emoji:    discordgo.ComponentEmoji{Name: "💵"},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{button}},
			},
		},
	})
}

// Pannello /setup cittadinanza
func sendCittadinanzaPanel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "🇮🇹 ANAGRAFE CENTRALE - ROME ITALY 2.0",
		Description: "Richiedi la tua Carta d'Identità per diventare un cittadino riconosciuto nel server.",
		Color:       0x3498DB,
	}

	button := discordgo.Button{
		Label:    "Richiedi Cittadinanza",
		Style:    discordgo.PrimaryButton,
		CustomID: "btn_richiedi_cittadinanza",
		Emoji:    discordgo.ComponentEmoji{Name: "🆔"},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{button}},
			},
		},
	})
}

// Pannello /setup registra-arresto
func sendRegistraArrestoPanel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "🚨 BANCA DATI POLIZIA DI STATO /
