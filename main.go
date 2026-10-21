package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Configurazione canali e ruoli del server
type ServerConfig struct {
	PanelChannelID string
	EsitiChannelID string
	AuthRoleID     string
}

// Struttura per le Tasse Generali di Base
type GeneralTaxes struct {
	Conto          int
	Veicoli        int
	Immobili       int
	IntervalDays   int    // Frequenza pagamento tasse in giorni
	CustomDeadline string // Note sulla scadenza (es. "Entro la domenica")
}

// Struttura per le Tasse Personalizzate per Singolo Utente
type UserTaxOverride struct {
	Conto    int
	Veicoli  int
	Immobili int
}

type BankAccount struct {
	OwnerName   string
	RobloxName  string
	BankName    string
	Balance     int
	LastPayday  time.Time
	LastTaxPaid time.Time // Traccia l'ultimo pagamento tasse effettuato
}

type AnagrafeRecord struct {
	UserID      string
	RobloxUser  string
	NomeRP      string
	CognomeRP   string
	DataNascita string
}

type PortoDarmiRecord struct {
	UserID      string
	RobloxUser  string
	NomeRP      string
	Motivazione string
}

type ImmatricolazioneRecord struct {
	UserID      string
	RobloxUser  string
	Targa       string
	ModelloAuto string
}

var serverConfigs = make(map[string]*ServerConfig)
var userBankAccounts = make(map[string]BankAccount)
var dbAnagrafe = make(map[string]AnagrafeRecord)
var dbPortoDarmi = make(map[string]PortoDarmiRecord)
var dbImmatricolazioni = make(map[string]ImmatricolazioneRecord)

// Configurazione Tasse Generali di default
var defaultTaxes = GeneralTaxes{
	Conto:          100,
	Veicoli:        200,
	Immobili:       500,
	IntervalDays:   7,
	CustomDeadline: "Ogni 7 giorni",
}

// Mappa per le tasse personalizzate: UserID -> UserTaxOverride
var userTaxOverrides = make(map[string]UserTaxOverride)

// Stipendio fisso a 12.000 Euro a settimana
var stipendioFisso = 12000

func getOrCreateConfig(key string) *ServerConfig {
	if serverConfigs[key] == nil {
		serverConfigs[key] = &ServerConfig{}
	}
	return serverConfigs[key]
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Bot Rome Italy 2.0 Online!")
		})
		log.Printf("Server HTTP avviato sulla porta %s", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Printf("Errore server HTTP: %v", err)
		}
	}()

	token := strings.TrimSpace(os.Getenv("DISCORD_TOKEN"))
	if token == "" {
		log.Println("⚠️ ERRORE: La variabile d'ambiente DISCORD_TOKEN non è impostata!")
	} else {
		if strings.HasPrefix(token, "Bot ") {
			token = strings.TrimPrefix(token, "Bot ")
		}

		dg, err := discordgo.New("Bot " + token)
		if err != nil {
			log.Printf("Errore durante la creazione della sessione Discord: %v", err)
		} else {
			dg.AddHandler(interactionHandler)

			err = dg.Open()
			if err != nil {
				log.Printf("Errore durante l'apertura della connessione a Discord: %v", err)
			} else {
				defer dg.Close()
				log.Println("✅ Bot 'Rome Italy 2.0' avviato con successo su Render!")
				registerCommands(dg)
			}
		}
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Arresto del bot...")
}

func registerCommands(s *discordgo.Session) {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "conto",
			Description: "Visualizza il saldo del tuo conto corrente bancario",
		},
		{
			Name:        "visualizza",
			Description: "Comandi di visualizzazione database per lo Staff",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "anagrafe",
					Description: "Visualizza tutte le carte d'identità registrate",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "porto-darmi",
					Description: "Visualizza tutte le richieste di porto d'armi",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "immatricolazioni",
					Description: "Visualizza tutti i veicoli immatricolati",
				},
			},
		},
		{
			Name:        "setup",
			Description: "Configura i pannelli interattivi e i canali del server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "server",
					Description: "Pannello generale per configurare canali (#) e ruoli (@) di ogni comando",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "apri-conto",
					Description: "Invia il pannello bancario per l'apertura del conto",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "busta-paga",
					Description: "Invia il pannello per la riscossione dello stipendio settimanale (€ 12.000)",
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
					Description: "Invia il pannello per la sanzione stradale (verbale multa)",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "tasse",
					Description: "Invia il pannello e gestisci le tasse statali, immobili, veicoli e conto",
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
		// Registrazione istantanea direttamente sul tuo server Discord specifico
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "1488262125708116198", cmd)
		if err != nil {
			log.Printf("Errore nella registrazione del comando '%s': %v", cmd.Name, err)
		}
	}
}

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

func handleSlashCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	if data.Name == "conto" {
		userID := i.Member.User.ID
		account, exists := userBankAccounts[userID]
		if !exists {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Non possiedi ancora un conto corrente registrato! Usa il pannello `/setup apri-conto` per aprirne uno.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🏛️ ESTRATTO CONTO BANCARIO",
			Color:       0x2ECC71,
			Description: fmt.Sprintf("**Utente Discord:** <@%s>\n**Intestatario RP:** %s\n**Utente Roblox:** `@%s`\n**Istituto Bancario:** %s\n💰 **Saldo Disponibile:** `€ %d`", userID, account.OwnerName, account.RobloxName, strings.ToUpper(account.BankName), account.Balance),
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if data.Name == "visualizza" {
		sub := data.Options[0].Name
		switch sub {
		case "anagrafe":
			text := "📋 **LISTA CITTADINANZA / ANAGRAFE REGISTRATA:**\n\n"
			if len(dbAnagrafe) == 0 {
				text += "*Nessuna carta d'identità registrata.*"
			} else {
				for _, rec := range dbAnagrafe {
					text += fmt.Sprintf("• **Discord:** <@%s> | **RP:** %s %s | **Roblox:** `@%s`\n", rec.UserID, rec.NomeRP, rec.CognomeRP, rec.RobloxUser)
				}
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: text, Flags: discordgo.MessageFlagsEphemeral},
			})

		case "porto-darmi":
			text := "🔫 **LISTA PORTO D'ARMI RICHIESTI:**\n\n"
			if len(dbPortoDarmi) == 0 {
				text += "*Nessuna richiesta di porto d'armi.*"
			} else {
				for _, rec := range dbPortoDarmi {
					text += fmt.Sprintf("• **Discord:** <@%s> | **RP:** %s | **Roblox:** `@%s` | **Motivazione:** %s\n", rec.UserID, rec.NomeRP, rec.RobloxUser, rec.Motivazione)
				}
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: text, Flags: discordgo.MessageFlagsEphemeral},
			})

		case "immatricolazioni":
			text := "🚗 **LISTA VEICOLI IMMATRICOLATI (PRA):**\n\n"
			if len(dbImmatricolazioni) == 0 {
				text += "*Nessun veicolo immatricolato.*"
			} else {
				for _, rec := range dbImmatricolazioni {
					text += fmt.Sprintf("• **Discord:** <@%s> | **Modello:** %s | **Targa:** `%s` | **Roblox:** `@%s`\n", rec.UserID, rec.ModelloAuto, rec.Targa, rec.RobloxUser)
				}
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: text, Flags: discordgo.MessageFlagsEphemeral},
			})
		}
		return
	}

	options := data.Options
	if len(options) == 0 {
		return
	}

	subCmd := options[0].Name

	switch subCmd {
	case "server":
		sendServerSetupPanel(s, i)
	case "tasse":
		sendTasseSetupPanel(s, i)
	default:
		sendPanelPreview(s, i, subCmd)
	}
}

func sendTasseSetupPanel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cfg := getOrCreateConfig("tasse")
	targetChannel := cfg.PanelChannelID
	var channelInfo string
	if targetChannel == "" {
		channelInfo = "⚠️ *Nessun canale impostato in `/setup server`. Verrà inviato in questo canale.*"
	} else {
		channelInfo = fmt.Sprintf("📍 *Canale di destinazione:* <#%s>", targetChannel)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🏛️ GESTIONE FISCALE E TASSE - ROME ITALY 2.0",
		Description: fmt.Sprintf("Usa i pulsanti sottostanti per pubblicare il pannello o configurare le tasse generali, i tempi e le tariffe personalizzate ad personam.\n\n%s", channelInfo),
		Color:       0x95A5A6,
	}

	btnInvia := discordgo.Button{
		Label:    "Invia Pannello nel Canale",
		Style:    discordgo.SuccessButton,
		CustomID: "send_panel_to_channel_tasse",
		Emoji:    &discordgo.ComponentEmoji{Name: "📨"},
	}

	btnConfigBase := discordgo.Button{
		Label:    "⚙️ Tasse Generali",
		Style:    discordgo.PrimaryButton,
		CustomID: "btn_config_tasse_generali",
		Emoji:    &discordgo.ComponentEmoji{Name: "⚙️"},
	}

	btnConfigTime := discordgo.Button{
		Label:    "⏱️ Imposta Tempo / Frequenza",
		Style:    discordgo.PrimaryButton,
		CustomID: "btn_config_tasse_time",
		Emoji:    &discordgo.ComponentEmoji{Name: "⏱️"},
	}

	btnConfigUser := discordgo.Button{
		Label:    "👤 Tassa Personalizzata Utente",
		Style:    discordgo.SecondaryButton,
		CustomID: "btn_config_tasse_user",
		Emoji:    &discordgo.ComponentEmoji{Name: "👤"},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{btnInvia, btnConfigBase, btnConfigTime, btnConfigUser}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func sendPanelPreview(s *discordgo.Session, i *discordgo.InteractionCreate, serviceKey string) {
	cfg := getOrCreateConfig(serviceKey)
	targetChannel := cfg.PanelChannelID
	var channelInfo string
	if targetChannel == "" {
		channelInfo = "⚠️ *Nessun canale impostato in `/setup server`. Verrà inviato in questo canale.*"
	} else {
		channelInfo = fmt.Sprintf("📍 *Canale di destinazione impostato:* <#%s>", targetChannel)
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📤 INVIA PANNELLO: %s", strings.ToUpper(serviceKey)),
		Description: fmt.Sprintf("Clicca sul pulsante sottostante per pubblicare ufficialmente il pannello nel canale configurato.\n\n%s", channelInfo),
		Color:       0x3498DB,
	}

	btnSend := discordgo.Button{
		Label:    "Invia Pannello nel Canale",
		Style:    discordgo.SuccessButton,
		CustomID: "send_panel_to_channel_" + serviceKey,
		Emoji:    &discordgo.ComponentEmoji{Name: "📨"},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{btnSend}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func sendServerSetupPanel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "⚙️ CONFIGURAZIONE GLOBALE COMANDI - ROME ITALY 2.0",
		Description: "Seleziona dal menu a tendina il comando che vuoi configurare per impostare i canali e i ruoli autorizzati:",
		Color:       0x0099FF,
	}

	selectMenu := discordgo.SelectMenu{
		CustomID:    "setup_server_select",
		Placeholder: "Scegli il comando da configurare...",
		Options: []discordgo.SelectMenuOption{
			{Label: "Cittadinanza & Anagrafe", Value: "cittadinanza", Description: "Imposta canali e ruoli per /cittadinanza", Emoji: &discordgo.ComponentEmoji{Name: "🇮🇹"}},
			{Label: "Porto d'Armi", Value: "porto-darmi", Description: "Imposta canali e ruoli per /porto-darmi", Emoji: &discordgo.ComponentEmoji{Name: "🔫"}},
			{Label: "Apertura Conto Bancario", Value: "apri-conto", Description: "Imposta canali e ruoli per /apri-conto", Emoji: &discordgo.ComponentEmoji{Name: "💳"}},
			{Label: "Riscossione Busta Paga", Value: "busta-paga", Description: "Imposta canali e ruoli per /busta-paga", Emoji: &discordgo.ComponentEmoji{Name: "💵"}},
			{Label: "Immatricolazione Veicoli", Value: "immatricolazione", Description: "Imposta canali e ruoli per /immatricolazione", Emoji: &discordgo.ComponentEmoji{Name: "🚗"}},
			{Label: "Registro Arresti FDO", Value: "registra-arresto", Description: "Imposta canali e ruoli per /registra-arresto", Emoji: &discordgo.ComponentEmoji{Name: "🚨"}},
			{Label: "Emissione Multe", Value: "emetti-multa", Description: "Imposta canali e ruoli per /emetti-multa", Emoji: &discordgo.ComponentEmoji{Name: "📄"}},
			{Label: "Tasse e Fisco", Value: "tasse", Description: "Imposta canali e ruoli per /tasse", Emoji: &discordgo.ComponentEmoji{Name: "🏛️"}},
			{Label: "Staff & Moderazione Panel", Value: "staff-panel", Description: "Imposta canali e ruoli per /staff-panel", Emoji: &discordgo.ComponentEmoji{Name: "🛡️"}},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{selectMenu}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func handleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	// Pressione bottone "Invia Pannello nel Canale"
	if strings.HasPrefix(customID, "send_panel_to_channel_") {
		serviceKey := strings.TrimPrefix(customID, "send_panel_to_channel_")

		cfg := getOrCreateConfig(serviceKey)
		targetChannel := cfg.PanelChannelID
		if targetChannel == "" {
			targetChannel = i.ChannelID
		}

		publishPublicPanel(s, targetChannel, serviceKey)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ **Pannello inviato con successo nel canale <#%s>!**", targetChannel),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if customID == "btn_config_tasse_generali" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_config_tasse_base",
				Title:    "Configura Tasse Generali Base",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_tax_conto", Label: "Tassa Conto Corrente (€)", Style: discordgo.TextInputShort, Value: fmt.Sprintf("%d", defaultTaxes.Conto), Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_tax_veicoli", Label: "Tassa Veicoli / Bollo Auto (€)", Style: discordgo.TextInputShort, Value: fmt.Sprintf("%d", defaultTaxes.Veicoli), Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_tax_immobili", Label: "Tassa Immobili / Proprietà (€)", Style: discordgo.TextInputShort, Value: fmt.Sprintf("%d", defaultTaxes.Immobili), Required: true}}},
				},
			},
		})
		return
	}

	if customID == "btn_config_tasse_time" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_config_tasse_time",
				Title:    "Imposta Tempo e Scadenza Tasse",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_tax_days", Label: "Ogni quanti giorni pagare le tasse?", Style: discordgo.TextInputShort, Value: fmt.Sprintf("%d", defaultTaxes.IntervalDays), Placeholder: "Es. 7", Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_tax_deadline_note", Label: "Note/Scadenza (es. Entro Domenica)", Style: discordgo.TextInputShort, Value: defaultTaxes.CustomDeadline, Placeholder: "Es. Entro la domenica", Required: true}}},
				},
			},
		})
		return
	}

	if customID == "btn_config_tasse_user" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_config_tasse_user",
				Title:    "Tassa Personalizzata per Utente",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_tax_user_mention", Label: "Utente Discord (@Menzione o ID)", Style: discordgo.TextInputShort, Placeholder: "Es. @Gigetto o ID", Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_tax_u_conto", Label: "Tassa Conto (€)", Style: discordgo.TextInputShort, Placeholder: "100", Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_tax_u_veicoli", Label: "Tassa Veicoli (€)", Style: discordgo.TextInputShort, Placeholder: "2000", Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_tax_u_immobili", Label: "Tassa Immobili (€)", Style: discordgo.TextInputShort, Placeholder: "5000", Required: true}}},
				},
			},
		})
		return
	}

	// Approvazioni Staff (Sì/No)
	if strings.HasPrefix(customID, "approve_yes_") || strings.HasPrefix(customID, "approve_no_") {
		parts := strings.Split(customID, "_")
		azione := parts[1]
		targetUserID := parts[2]

		originalEmbed := i.Message.Embeds[0]
		newEmbed := *originalEmbed

		if azione == "yes" {
			newEmbed.Color = 0x2ECC71
			newEmbed.Title = "✅ RICHIESTA APPROVATA DALLO STAFF"
			newEmbed.Description += fmt.Sprintf("\n\n**Stato:** Approvato da <@%s>", i.Member.User.ID)
		} else {
			newEmbed.Color = 0xE74C3C
			newEmbed.Title = "❌ RICHIESTA RIFIUTATA DALLO STAFF"
			newEmbed.Description += fmt.Sprintf("\n\n**Stato:** Rifiutato da <@%s>", i.Member.User.ID)
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{&newEmbed},
				Components: []discordgo.MessageComponent{},
			},
		})

		privateChannel, err := s.UserChannelCreate(targetUserID)
		if err == nil {
			if azione == "yes" {
				s.ChannelMessageSend(privateChannel.ID, "🎉 Salve, la tua richiesta inviata nel server è stata **APPROVATA** dallo staff!")
			} else {
				s.ChannelMessageSend(privateChannel.ID, "❌ Salve, la tua richiesta inviata nel server è stata **RIFIUTATA** dallo staff.")
			}
		}
		return
	}

	if customID == "setup_server_select" {
		selectedCategory := "cittadinanza"
		if len(i.MessageComponentData().Values) > 0 {
			selectedCategory = i.MessageComponentData().Values[0]
		}

		channelSelectPanel := discordgo.SelectMenu{
			CustomID:    "selchanpanel_" + selectedCategory,
			Placeholder: "Seleziona il canale del comando (#)...",
			MenuType:    discordgo.ChannelSelectMenu,
			ChannelTypes: []discordgo.ChannelType{
				discordgo.ChannelTypeGuildText,
			},
		}

		channelSelectEsiti := discordgo.SelectMenu{
			CustomID:    "selchanesiti_" + selectedCategory,
			Placeholder: "Seleziona il canale dei risultati/esiti (#)...",
			MenuType:    discordgo.ChannelSelectMenu,
			ChannelTypes: []discordgo.ChannelType{
				discordgo.ChannelTypeGuildText,
			},
		}

		roleSelectAuth := discordgo.SelectMenu{
			CustomID:    "selroleauth_" + selectedCategory,
			Placeholder: "Seleziona il ruolo autorizzato (@)...",
			MenuType:    discordgo.RoleSelectMenu,
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("🛠️ **Configurazione per: %s**\nScegli dai menu a tendina i canali (`#`) e il ruolo (`@`):", strings.ToUpper(selectedCategory)),
				Flags:   discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{channelSelectPanel}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{channelSelectEsiti}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{roleSelectAuth}},
				},
			},
		})
		return
	}

	if strings.HasPrefix(customID, "selchanpanel_") || strings.HasPrefix(customID, "selchanesiti_") || strings.HasPrefix(customID, "selroleauth_") {
		values := i.MessageComponentData().Values
		if len(values) > 0 {
			valoreScelto := values[0]

			var servizioKey string
			if strings.HasPrefix(customID, "selchanpanel_") {
				servizioKey = strings.TrimPrefix(customID, "selchanpanel_")
			} else if strings.HasPrefix(customID, "selchanesiti_") {
				servizioKey = strings.TrimPrefix(customID, "selchanesiti_")
			} else if strings.HasPrefix(customID, "selroleauth_") {
				servizioKey = strings.TrimPrefix(customID, "selroleauth_")
			}

			cfg := getOrCreateConfig(servizioKey)
			if strings.HasPrefix(customID, "selchanpanel_") {
				cfg.PanelChannelID = valoreScelto
			} else if strings.HasPrefix(customID, "selchanesiti_") {
				cfg.EsitiChannelID = valoreScelto
			} else if strings.HasPrefix(customID, "selroleauth_") {
				cfg.AuthRoleID = valoreScelto
			}
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "✅ **Selezione salvata con successo nel sistema!**",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	switch customID {
	case "btn_paga_tasse":
		userID := i.Member.User.ID
		account, exists := userBankAccounts[userID]
		if !exists {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Devi prima aprire un conto corrente bancario usando `/setup apri-conto` per poter versare le tasse!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		taxInterval := time.Duration(defaultTaxes.IntervalDays) * 24 * time.Hour
		if taxInterval > 0 && time.Since(account.LastTaxPaid) < taxInterval {
			rimanente := taxInterval - time.Since(account.LastTaxPaid)
			giorni := int(rimanente.Hours() / 24)
			ore := int(int(rimanente.Hours()) % 24)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("⏳ **Hai già saldato le tasse per questo periodo!**\nPotrai versarle nuovamente tra **%d giorni e %d ore**.\n📌 *Scadenza:* %s", giorni, ore, defaultTaxes.CustomDeadline),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		taxConto := defaultTaxes.Conto
		taxVeicoli := defaultTaxes.Veicoli
		taxImmobili := defaultTaxes.Immobili
		isPersonalized := false

		if override, hasOverride := userTaxOverrides[userID]; hasOverride {
			taxConto = override.Conto
			taxVeicoli = override.Veicoli
			taxImmobili = override.Immobili
			isPersonalized = true
		}

		totaleTasse := taxConto + taxVeicoli + taxImmobili

		if account.Balance < totaleTasse {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("⚠️ **Saldo insufficiente sul conto bancario!** Totale tasse da saldare: **€ %d**, Saldo attuale: `€ %d`.", totaleTasse, account.Balance),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		account.Balance -= totaleTasse
		account.LastTaxPaid = time.Now()
		userBankAccounts[userID] = account

		tipoAliquota := "Generale Standard"
		if isPersonalized {
			tipoAliquota = "Personalizzata Ad Personam"
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("🏛️ **Tasse Versate con Successo!**\n• Conto Corrente: € %d\n• Veicoli: € %d\n• Immobili: € %d\n💰 **Totale Detratto:** `€ %d` | Nuovo Saldo: `€ %d`", taxConto, taxVeicoli, taxImmobili, totaleTasse, account.Balance),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		cfg := getOrCreateConfig("tasse")
		targetChannel := cfg.EsitiChannelID
		if targetChannel == "" {
			targetChannel = i.ChannelID
		}

		embedEsito := &discordgo.MessageEmbed{
			Title: "🏛️ RICEVUTA PAGAMENTO TASSE STATALI",
			Color: 0x2ECC71,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Cittadino Discord", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
				{Name: "Tipo Aliquota", Value: tipoAliquota, Inline: true},
				{Name: "Tassa Conto", Value: fmt.Sprintf("€ %d", taxConto), Inline: true},
				{Name: "Tassa Veicoli", Value: fmt.Sprintf("€ %d", taxVeicoli), Inline: true},
				{Name: "Tassa Immobili", Value: fmt.Sprintf("€ %d", taxImmobili), Inline: true},
				{Name: "Totale Versato", Value: fmt.Sprintf("`€ %d`", totaleTasse), Inline: true},
			},
		}

		msgData := &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embedEsito}}
		s.ChannelMessageSendComplex(targetChannel, msgData)

	case "btn_apri_conto":
		selectBanche := discordgo.SelectMenu{
			CustomID:    "select_banca_choice",
			Placeholder: "Seleziona il tuo istituto bancario...",
			Options: []discordgo.SelectMenuOption{
				{Label: "Banca Sella", Value: "banca_sella", Emoji: &discordgo.ComponentEmoji{Name: "🏛️"}},
				{Label: "Poste Italiane", Value: "poste", Emoji: &discordgo.ComponentEmoji{Name: "📮"}},
				{Label: "Intesa Sanpaolo", Value: "intesa", Emoji: &discordgo.ComponentEmoji{Name: "🏦"}},
				{Label: "UniCredit", Value: "unicredit", Emoji: &discordgo.ComponentEmoji{Name: "🔴"}},
				{Label: "BPER Banca", Value: "bper", Emoji: &discordgo.ComponentEmoji{Name: "🟦"}},
				{Label: "Banco BPM", Value: "bpm", Emoji: &discordgo.ComponentEmoji{Name: "🟢"}},
				{Label: "Revolut", Value: "revolut", Emoji: &discordgo.ComponentEmoji{Name: "💳"}},
			},
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Scegli la banca con cui desideri aprire il conto corrente (riceverai subito **€ 7.000** di bonus):",
				Flags:   discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{selectBanche}}},
			},
		})

	case "select_banca_choice":
		selectedBanca := "banca_generica"
		if len(i.MessageComponentData().Values) > 0 {
			selectedBanca = i.MessageComponentData().Values[0]
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_conferma_conto_" + selectedBanca,
				Title:    "Apertura Conto Corrente",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_nome_cognome_rp", Label: "Nome e Cognome RP", Style: discordgo.TextInputShort, Placeholder: "Es. Mario Rossi", Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_roblox_username", Label: "Username Roblox", Style: discordgo.TextInputShort, Placeholder: "Es. RobloxUser123", Required: true}}},
				},
			},
		})

	case "btn_richiedi_cittadinanza":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_cittadinanza",
				Title:    "Registrazione Carta d'Identita",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_nome_rp", Label: "Nome RP", Style: discordgo.TextInputShort, Placeholder: "Es. Mario", Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_cognome_rp", Label: "Cognome RP", Style: discordgo.TextInputShort, Placeholder: "Es. Rossi", Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_roblox_user", Label: "Username Roblox", Style: discordgo.TextInputShort, Placeholder: "Es. RobloxPlayer1", Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_data_nascita", Label: "Data di Nascita RP", Style: discordgo.TextInputShort, Placeholder: "GG/MM/AAAA", Required: true}}},
				},
			},
		})

	case "btn_richiedi_porto_darmi":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_porto_darmi",
				Title:    "Richiesta Licenza Porto d'Armi",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_nome_cognome_pda", Label: "Nome e Cognome RP", Style: discordgo.TextInputShort, Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_roblox_pda", Label: "Username Roblox", Style: discordgo.TextInputShort, Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_motivazione_pda", Label: "Motivazione Richiesta", Style: discordgo.TextInputParagraph, Required: true}}},
				},
			},
		})

	case "btn_immatricolazione":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_immatricolazione",
				Title:    "Immatricolazione Veicolo (PRA)",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_roblox_imm", Label: "Username Roblox", Style: discordgo.TextInputShort, Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_modello_auto", Label: "Modello Veicolo", Style: discordgo.TextInputShort, Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_targa", Label: "Targa Desiderata", Style: discordgo.TextInputShort, Required: true}}},
				},
			},
		})

	case "btn_registra_arresto":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_registra_arresto",
				Title:    "Verbale d'Arresto",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_utente_arrestato", Label: "Utente Roblox Arrestato", Style: discordgo.TextInputShort, Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_reati", Label: "Reati Commessi", Style: discordgo.TextInputParagraph, Placeholder: "Inseguimento...", Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_prigione", Label: "Anni o Mesi di Prigione", Style: discordgo.TextInputShort, Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_cauzione", Label: "Cauzione In Euro", Style: discordgo.TextInputShort, Placeholder: "5000", Required: false}}},
				},
			},
		})

	case "btn_emetti_multa":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: "modal_emetti_multa",
				Title:    "Sanzione Stradale / Verbale Multa",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_utente_multato", Label: "Utente Roblox Multato", Style: discordgo.TextInputShort, Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_importo_multa", Label: "Importo Multa in Euro (€)", Style: discordgo.TextInputShort, Placeholder: "150", Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_motivo_multa", Label: "Motivazione / Infrazione", Style: discordgo.TextInputParagraph, Placeholder: "Eccesso di velocità...", Required: true}}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "input_targa_multa", Label: "Targa Veicolo (opzionale)", Style: discordgo.TextInputShort, Required: false}}},
				},
			},
		})

	case "btn_riscuoti_stipendio":
		userID := i.Member.User.ID
		account, exists := userBankAccounts[userID]
		if !exists {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Devi prima aprire un conto corrente bancario usando il pannello `/setup apri-conto` per poter riscuotere la busta paga!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		if time.Since(account.LastPayday) < 7*24*time.Hour {
			rimanente := (7 * 24 * time.Hour) - time.Since(account.LastPayday)
			giorni := int(rimanente.Hours() / 24)
			ore := int(int(rimanente.Hours()) % 24)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("⏳ **Hai già riscosso la busta paga questa settimana!** Potrai riscuoterla nuovamente tra **%d giorni e %d ore**.", giorni, ore),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		account.Balance += stipendioFisso
		account.LastPayday = time.Now()
		userBankAccounts[userID] = account

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ **Busta Paga Riscossa con Successo!** Ti sono stati accreditati **€ 12.000** sul tuo conto corrente. Saldo totale attuale: `€ %d`", account.Balance),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

func publishPublicPanel(s *discordgo.Session, channelID string, serviceKey string) {
	switch serviceKey {
	case "apri-conto":
		embed := &discordgo.MessageEmbed{
			Title:       "🏛️ APERTURA CONTO CORRENTE - ROME ITALY 2.0",
			Description: "Clicca sul bottone qui sotto per scegliere il tuo istituto bancario e ricevere **€ 7.000** di bonus benvenuto sul tuo conto!",
			Color:       0x2ECC71,
		}
		btn := discordgo.Button{Label: "Apri Conto Corrente", Style: discordgo.SuccessButton, CustomID: "btn_apri_conto", Emoji: &discordgo.ComponentEmoji{Name: "💳"}}
		s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{btn}}}})

	case "porto-darmi":
		embed := &discordgo.MessageEmbed{
			Title:       "🔫 RICHIESTA PORTO D'ARMI - ROME ITALY 2.0",
			Description: "Clicca sul bottone per inviare la richiesta di licenza porto d'armi alle autorità competenti.",
			Color:       0xE67E22,
		}
		btn := discordgo.Button{Label: "Richiedi Porto d'Armi", Style: discordgo.PrimaryButton, CustomID: "btn_richiedi_porto_darmi", Emoji: &discordgo.ComponentEmoji{Name: "📜"}}
		s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{btn}}}})

	case "busta-paga":
		embed := &discordgo.MessageEmbed{
			Title:       "💼 RISCOSSIONE BUSTA PAGA - ROME ITALY 2.0",
			Description: "Clicca sul bottone qui sotto per riscuotere la tua busta paga settimanale di **€ 12.000** sul tuo conto corrente!",
			Color:       0xF1C40F,
		}
		btn := discordgo.Button{Label: "Riscuoti Busta Paga (€ 12.000)", Style: discordgo.PrimaryButton, CustomID: "btn_riscuoti_stipendio", Emoji: &discordgo.ComponentEmoji{Name: "💵"}}
		s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{btn}}}})

	case "cittadinanza":
		embed := &discordgo.MessageEmbed{
			Title:       "🇮🇹 ANAGRAFE CENTRALE - ROME ITALY 2.0",
			Description: "Richiedi la tua Carta d'Identità e la Cittadinanza ufficiali.",
			Color:       0x3498DB,
		}
		btn := discordgo.Button{Label: "Richiedi Cittadinanza", Style: discordgo.PrimaryButton, CustomID: "btn_richiedi_cittadinanza", Emoji: &discordgo.ComponentEmoji{Name: "🆔"}}
		s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{btn}}}})

	case "immatricolazione":
		embed := &discordgo.MessageEmbed{
			Title:       "🚗 UFFICIO PRA - IMMATRICOLAZIONE VEICOLI",
			Description: "Clicca sul bottone per registrare e immatricolare ufficialmente il tuo veicolo nel server.",
			Color:       0x9B59B6,
		}
		btn := discordgo.Button{Label: "Immatricola Veicolo", Style: discordgo.PrimaryButton, CustomID: "btn_immatricolazione", Emoji: &discordgo.ComponentEmoji{Name: "🚘"}}
		s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{btn}}}})

	case "registra-arresto":
		embed := &discordgo.MessageEmbed{
			Title:       "🚨 BANCA DATI POLIZIA DI STATO / CARABINIERI",
			Description: "Compila il verbale dei fermi e degli arresti eseguiti durante il servizio.",
			Color:       0xE74C3C,
		}
		btn := discordgo.Button{Label: "Registra Arresto", Style: discordgo.DangerButton, CustomID: "btn_registra_arresto", Emoji: &discordgo.ComponentEmoji{Name: "📝"}}
		s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{btn}}}})

	case "emetti-multa":
		embed := &discordgo.MessageEmbed{
			Title:       "📄 POLIZIA LOCALE - EMISSIONE VERBALE MULTA",
			Description: "Clicca sul bottone per emettere una sanzione stradale nei confronti di un cittadino.",
			Color:       0xE74C3C,
		}
		btn := discordgo.Button{Label: "Emetti Multa", Style: discordgo.DangerButton, CustomID: "btn_emetti_multa", Emoji: &discordgo.ComponentEmoji{Name: "📄"}}
		s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{btn}}}})

	case "tasse":
		embed := &discordgo.MessageEmbed{
			Title:       "🏛️ AGENZIA DELLE ENTRATE - PAGAMENTO TASSE STATALI",
			Description: fmt.Sprintf("Clicca sul bottone sottostante per effettuare il versamento delle imposte statali sul conto corrente, sugli immobili e sui veicoli.\n\n📌 **Scadenza / Frequenza:** %s", defaultTaxes.CustomDeadline),
			Color:       0x95A5A6,
		}
		btn := discordgo.Button{Label: "Paga Tasse Statali", Style: discordgo.SuccessButton, CustomID: "btn_paga_tasse", Emoji: &discordgo.ComponentEmoji{Name: "💸"}}
		s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{btn}}}})
	}
}

func handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID

	if customID == "modal_config_tasse_base" {
		data := i.ModalSubmitData()
		fmt.Sscanf(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value, "%d", &defaultTaxes.Conto)
		fmt.Sscanf(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value, "%d", &defaultTaxes.Veicoli)
		fmt.Sscanf(data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value, "%d", &defaultTaxes.Immobili)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ **Tasse Generali Aggiornate!**\n• Conto: € %d\n• Veicoli: € %d\n• Immobili: € %d", defaultTaxes.Conto, defaultTaxes.Veicoli, defaultTaxes.Immobili),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if customID == "modal_config_tasse_time" {
		data := i.ModalSubmitData()
		fmt.Sscanf(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value, "%d", &defaultTaxes.IntervalDays)
		defaultTaxes.CustomDeadline = data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ **Tempo Tasse Aggiornato!**\n• Frequenza Pagamento: Ogni **%d giorni**\n• Note Scadenza: *%s*", defaultTaxes.IntervalDays, defaultTaxes.CustomDeadline),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if customID == "modal_config_tasse_user" {
		data := i.ModalSubmitData()
		userMention := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

		targetUserID := strings.TrimPrefix(userMention, "<@")
		targetUserID = strings.TrimPrefix(targetUserID, "!")
		targetUserID = strings.TrimSuffix(targetUserID, ">")
		targetUserID = strings.TrimSpace(targetUserID)

		var c, v, im int
		fmt.Sscanf(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value, "%d", &c)
		fmt.Sscanf(data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value, "%d", &v)
		fmt.Sscanf(data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value, "%d", &im)

		userTaxOverrides[targetUserID] = UserTaxOverride{
			Conto:    c,
			Veicoli:  v,
			Immobili: im,
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ **Tassa Personalizzata impostata per <@%s>!**\n• Conto: € %d\n• Veicoli: € %d\n• Immobili: € %d", targetUserID, c, v, im),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if strings.HasPrefix(customID, "modal_conferma_conto_") {
		banca := strings.TrimPrefix(customID, "modal_conferma_conto_")
		data := i.ModalSubmitData()
		nomeRP := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		robloxUser := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		userID := i.Member.User.ID

		userBankAccounts[userID] = BankAccount{
			OwnerName:  nomeRP,
			RobloxName: robloxUser,
			BankName:   banca,
			Balance:    7000,
			LastPayday: time.Time{},
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "🎉 **Conto Corrente Aperto con Successo!** Ti sono stati accreditati **€ 7.000** di bonus benvenuto. Verifica il saldo con `/conto`.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		cfg := getOrCreateConfig("apri-conto")
		targetChannel := cfg.EsitiChannelID
		if targetChannel == "" {
			targetChannel = i.ChannelID
		}

		embedEsito := &discordgo.MessageEmbed{
			Title:       "💳 NUOVA RICHIESTA APERTURA CONTO BANCARIO",
			Color:       0x2ECC71,
			Description: fmt.Sprintf("**Richiedente Discord:** <@%s>\n**Utente Roblox:** `@%s`\n**Nome & Cognome RP:** %s\n**Banca:** %s\n\n*Saldo accreditato:* € 7.000", userID, robloxUser, nomeRP, strings.ToUpper(banca)),
		}

		btnYes := discordgo.Button{Label: "Accetta (Sì)", Style: discordgo.SuccessButton, CustomID: "approve_yes_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "✅"}}
		btnNo := discordgo.Button{Label: "Rifiuta (No)", Style: discordgo.DangerButton, CustomID: "approve_no_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "❌"}}

		msgData := &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embedEsito},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{btnYes, btnNo}},
			},
		}

		s.ChannelMessageSendComplex(targetChannel, msgData)
		return
	}

	switch customID {
	case "modal_porto_darmi":
		data := i.ModalSubmitData()
		nomeRP := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		robloxUser := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		motivazione := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		userID := i.Member.User.ID

		dbPortoDarmi[userID] = PortoDarmiRecord{
			UserID:      userID,
			RobloxUser:  robloxUser,
			NomeRP:      nomeRP,
			Motivazione: motivazione,
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📜 **Richiesta Porto d'Armi Inviata!** Sarà valutata dalle autorità.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		cfg := getOrCreateConfig("porto-darmi")
		targetChannel := cfg.EsitiChannelID
		if targetChannel == "" {
			targetChannel = i.ChannelID
		}

		embedPda := &discordgo.MessageEmbed{
			Title: "🔫 RICHIESTA LICENZA PORTO D'ARMI",
			Color: 0xE67E22,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Richiedente Discord", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
				{Name: "Utente Roblox", Value: fmt.Sprintf("`@%s`", robloxUser), Inline: true},
				{Name: "Nome & Cognome RP", Value: nomeRP, Inline: true},
				{Name: "Motivazione", Value: motivazione, Inline: false},
			},
		}

		btnYes := discordgo.Button{Label: "Accetta (Sì)", Style: discordgo.SuccessButton, CustomID: "approve_yes_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "✅"}}
		btnNo := discordgo.Button{Label: "Rifiuta (No)", Style: discordgo.DangerButton, CustomID: "approve_no_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "❌"}}

		msgData := &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embedPda},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{btnYes, btnNo}},
			},
		}

		s.ChannelMessageSendComplex(targetChannel, msgData)

	case "modal_cittadinanza":
		data := i.ModalSubmitData()
		nomeRP := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		cognomeRP := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		robloxUser := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		dataNascita := data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		userID := i.Member.User.ID

		dbAnagrafe[userID] = AnagrafeRecord{
			UserID:      userID,
			RobloxUser:  robloxUser,
			NomeRP:      nomeRP,
			CognomeRP:   cognomeRP,
			DataNascita: dataNascita,
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "🇮🇹 **Richiesta Cittadinanza Registrata!** Inviata nel canale esiti.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		cfg := getOrCreateConfig("cittadinanza")
		targetChannel := cfg.EsitiChannelID
		if targetChannel == "" {
			targetChannel = i.ChannelID
		}

		embedAnagrafe := &discordgo.MessageEmbed{
			Title: "🇮🇹 NUOVA RICHIESTA CITTADINANZA / CARTA D'IDENTITÀ",
			Color: 0x3498DB,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Richiedente Discord", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
				{Name: "Utente Roblox", Value: fmt.Sprintf("`@%s`", robloxUser), Inline: true},
				{Name: "Nome RP", Value: nomeRP, Inline: true},
				{Name: "Cognome RP", Value: cognomeRP, Inline: true},
				{Name: "Data di Nascita", Value: dataNascita, Inline: true},
			},
		}

		btnYes := discordgo.Button{Label: "Accetta (Sì)", Style: discordgo.SuccessButton, CustomID: "approve_yes_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "✅"}}
		btnNo := discordgo.Button{Label: "Rifiuta (No)", Style: discordgo.DangerButton, CustomID: "approve_no_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "❌"}}

		msgData := &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embedAnagrafe},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{btnYes, btnNo}},
			},
		}

		s.ChannelMessageSendComplex(targetChannel, msgData)

	case "modal_immatricolazione":
		data := i.ModalSubmitData()
		robloxUser := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		modelloAuto := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		targa := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		userID := i.Member.User.ID

		dbImmatricolazioni[targa] = ImmatricolazioneRecord{
			UserID:      userID,
			RobloxUser:  robloxUser,
			Targa:       targa,
			ModelloAuto: modelloAuto,
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "🚗 **Immatricolazione Inviata!** Inserita nel registro PRA.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		cfg := getOrCreateConfig("immatricolazione")
		targetChannel := cfg.EsitiChannelID
		if targetChannel == "" {
			targetChannel = i.ChannelID
		}

		embedImm := &discordgo.MessageEmbed{
			Title: "🚗 NUOVA RICHIESTA IMMATRICOLAZIONE VEICOLO",
			Color: 0x9B59B6,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Richiedente Discord", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
				{Name: "Utente Roblox", Value: fmt.Sprintf("`@%s`", robloxUser), Inline: true},
				{Name: "Modello Veicolo", Value: modelloAuto, Inline: true},
				{Name: "Targa Richiesta", Value: fmt.Sprintf("`%s`", targa), Inline: true},
			},
		}

		btnYes := discordgo.Button{Label: "Accetta (Sì)", Style: discordgo.SuccessButton, CustomID: "approve_yes_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "✅"}}
		btnNo := discordgo.Button{Label: "Rifiuta (No)", Style: discordgo.DangerButton, CustomID: "approve_no_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "❌"}}

		msgData := &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embedImm},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{btnYes, btnNo}},
			},
		}

		s.ChannelMessageSendComplex(targetChannel, msgData)

	case "modal_registra_arresto":
		data := i.ModalSubmitData()
		utenteArrestato := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		reati := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		prigione := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		cauzione := data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		userID := i.Member.User.ID

		cfg := getOrCreateConfig("registra-arresto")
		targetChannel := cfg.EsitiChannelID
		if targetChannel == "" {
			targetChannel = i.ChannelID
		}

		embedArresto := &discordgo.MessageEmbed{
			Title: "🚨 VERBALE D'ARRESTO UFFICIALE",
			Color: 0xE74C3C,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Agente Discord", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
				{Name: "Arrestato Roblox", Value: fmt.Sprintf("`@%s`", utenteArrestato), Inline: true},
				{Name: "Pena", Value: prigione, Inline: true},
				{Name: "Cauzione", Value: cauzione, Inline: true},
				{Name: "Reati", Value: reati, Inline: false},
			},
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "🚨 **Verbale Registrato!** Inviato nel canale del registro ufficiale.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		btnYes := discordgo.Button{Label: "Approva Verbale (Sì)", Style: discordgo.SuccessButton, CustomID: "approve_yes_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "✅"}}
		btnNo := discordgo.Button{Label: "Annulla Verbale (No)", Style: discordgo.DangerButton, CustomID: "approve_no_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "❌"}}

		msgData := &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embedArresto},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{btnYes, btnNo}},
			},
		}

		s.ChannelMessageSendComplex(targetChannel, msgData)

	case "modal_emetti_multa":
		data := i.ModalSubmitData()
		utenteMultato := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		importo := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		motivo := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		targa := data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		userID := i.Member.User.ID

		cfg := getOrCreateConfig("emetti-multa")
		targetChannel := cfg.EsitiChannelID
		if targetChannel == "" {
			targetChannel = i.ChannelID
		}

		if targa == "" {
			targa = "N/D"
		}

		embedMulta := &discordgo.MessageEmbed{
			Title: "📄 SANZIONE STRADALE / VERBALE MULTA",
			Color: 0xE74C3C,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Agente Discord", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
				{Name: "Cittadino Multato (Roblox)", Value: fmt.Sprintf("`@%s`", utenteMultato), Inline: true},
				{Name: "Importo Multa", Value: fmt.Sprintf("€ %s", importo), Inline: true},
				{Name: "Targa Veicolo", Value: fmt.Sprintf("`%s`", targa), Inline: true},
				{Name: "Infrazione / Motivazione", Value: motivo, Inline: false},
			},
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📄 **Verbale di Multa Emetto e Registrato con Successo!**",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		btnYes := discordgo.Button{Label: "Approva Multa (Sì)", Style: discordgo.SuccessButton, CustomID: "approve_yes_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "✅"}}
		btnNo := discordgo.Button{Label: "Annulla Multa (No)", Style: discordgo.DangerButton, CustomID: "approve_no_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "❌"}}

		msgData := &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embedMulta},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{btnYes, btnNo}},
			},
		}

		s.ChannelMessageSendComplex(targetChannel, msgData)
	}
}
