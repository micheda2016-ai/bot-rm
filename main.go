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

func main() {
	// Server di keep-alive per Render
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "Hamburg Gang Bot Online") })
		port := os.Getenv("PORT")
		if port == "" { port = "8080" }
		http.ListenAndServe(":"+port, nil)
	}()

	token := os.Getenv("DISCORD_TOKEN")
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Errore sessione: %v", err)
	}

	// ⚠️ CONFIGURAZIONE ID (Assicurati che l'ID del Server sia corretto!)
	serverID          := "1495170590947016996" 
	capoGangRole      := "1502996558340296754" 
	reclutatoriRole1  := "1495179869061906602" 
	reclutatoriRole2  := "1495179516094709850" 
	
	// Grado autorizzato per eseguire Ban, Warn e Blacklist
	staffAutorizzatoRole := "1526849011510673638"

	// 1. DEFINIZIONE DEI COMANDI SLASH (Nativi di Discord)
	commands := []*discordgo.ApplicationCommand{
		{Name: "setup-gang", Description: "Invia il pannello di reclutamento e ticket della gang"},
		{Name: "rinforzi", Description: "Richiedi rinforzi immediati sul campo!"},
		{Name: "arsenale", Description: "Visualizza le regole sull'arsenale e l'equipaggiamento"},
		{Name: "colpo", Description: "Pianifica un'azione o rapina di gruppo"},
		
		// COMANDO BAN
		{
			Name:        "ban-gang",
			Description: "Banna un utente dalla gang (Solo Staff autorizzato)",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "nome-discord", Description: "Seleziona l'utente Discord", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "nome-roblox", Description: "Inserisci il nome Roblox", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivo del ban", Required: true},
			},
		},
		
		// COMANDO WARN
		{
			Name:        "warn-gang",
			Description: "Assegna un ammonimento (warn) a un membro (Solo Staff autorizzato)",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "nome-discord", Description: "Seleziona l'utente Discord da ammonire", Required: true},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "warn",
					Description: "Numero del warn (da 1 a 3)",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Warn 1", Value: 1},
						{Name: "Warn 2", Value: 2},
						{Name: "Warn 3 (Espulsione)", Value: 3},
					},
				},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivo del warn", Required: true},
			},
		},
		
		// COMANDO BLACKLIST
		{
			Name:        "blacklist-gang",
			Description: "Inserisce un utente nella blacklist della gang (Solo Staff autorizzato)",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "nome-discord", Description: "Seleziona l'utente Discord", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "nome-roblox", Description: "Inserisci il nome Roblox", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivo della blacklist", Required: true},
			},
		},
	}

	// Funzione di supporto per verificare se l'utente ha il ruolo Staff richiesto
	hasStaffRole := func(member *discordgo.Member) bool {
		for _, rID := range member.Roles {
			if rID == staffAutorizzatoRole {
				return true
			}
		}
		return false
	}

	// 2. COMANDI DI TESTO (!reclama e !chiudi)
	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot { return }
		content := strings.ToLower(m.Content)

		if content == "!reclama" {
			s.ChannelEditComplex(m.ChannelID, &discordgo.ChannelEdit{
				PermissionOverwrites: []*discordgo.PermissionOverwrite{
					{ID: m.GuildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: 1024},
					{ID: m.Author.ID, Type: discordgo.PermissionOverwriteTypeMember, Allow: 3072},
					{ID: capoGangRole, Type: discordgo.PermissionOverwriteTypeRole, Allow: 3072},
				},
			})
			s.ChannelMessageSend(m.ChannelID, "✅ **PROVINO/TICKET PRESO IN CARICO**\nGestito da: " + m.Author.Mention() + "\nAccesso riservato all'operatore e ai Capi della Gang.")
		}

		if content == "!chiudi" {
			s.ChannelMessageSend(m.ChannelID, "🔒 **CHIUSURA CANALE**\nL'archivio verrà eliminato tra 5 secondi...")
			time.Sleep(5 * time.Second)
			s.ChannelDelete(m.ChannelID)
		}
	})

	// 3. GESTIONE INTERAZIONI (Slash, Menu e Pulsanti)
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		
		// Gestione comandi Slash
		if i.Type == discordgo.InteractionApplicationCommand {
			data := i.ApplicationCommandData()
			options := data.Options
			
			// Trasformiamo le opzioni in una mappa
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			switch data.Name {
			case "setup-gang":
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: 4,
					Data: &discordgo.InteractionResponseData{
						Content: "⚓ **HAMBURG SYNDICATE - QUARTIERE GENERALE**\nUsa il menu sottostante per fare richiesta di ingresso o parlare con i Capi.",
						Components: []discordgo.MessageComponent{
							discordgo.ActionsRow{Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									CustomID: "ticket_gang",
									Placeholder: "Seleziona un'opzione...",
									Options: []discordgo.SelectMenuOption{
										{Label: "Richiesta Provino (Arruolamento)", Value: "arruolamento"},
										{Label: "Parla con la Dirigenza (Ticket)", Value: "dirigenza"},
									},
								},
							}},
						},
					},
				})

			case "rinforzi":
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: 4,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("🚨 **RICHIESTA RINFORZI URGENZIALE!**\nInviata da: %s\n\n⚠️ **TUTTI I MEMBRI DELLA GANG SONO PREGATI DI ENTRARE IN FREQUENZA E DIRIGERSI SUL POSTO!**", i.Member.User.Mention()),
					},
				})

			case "arsenale":
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: 4,
					Data: &discordgo.InteractionResponseData{
						Content: "🔫 **REGOLAMENTO ARSENALE GANG**\n1. Ogni membro deve essere equipaggiato prima di uscire in pattuglia.\n2. Vietato vendere o regalare le armi della gang a esterni.\n3. Rifornire i depositi dopo ogni colpo andato a segno.",
					},
				})

			case "colpo":
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: 4,
					Data: &discordgo.InteractionResponseData{
						Content: "💰 **PIANIFICAZIONE COLPO IN CORSO**\nUnisciti alla chiamata vocale 'Pianificazione' per ricevere i dettagli dell'azione coordinata.",
					},
				})

			case "ban-gang":
				// Controllo Grado
				if !hasStaffRole(i.Member) {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: 4,
						Data: &discordgo.InteractionResponseData{
							Content: "❌ **Non hai l'autorizzazione di Staff per usare questo comando.**",
							Flags:   64, // Messaggio visibile solo a chi lo digita (Episodico)
						},
					})
					return
				}

				discordUser := optionMap["nome-discord"].UserValue(s)
				robloxName := optionMap["nome-roblox"].StringValue()
				motivo := optionMap["motivo"].StringValue()

				messaggio := fmt.Sprintf(
					"🔨 **BAN EMESSO**\n\n• **Nome Discord:** %s\n• **Nome Roblox:** `%s`\n• **Motivo:** %s\n• **Eseguito da:** %s",
					discordUser.Mention(), robloxName, motivo, i.Member.User.Mention(),
				)

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: 4,
					Data: &discordgo.InteractionResponseData{Content: messaggio},
				})

			case "warn-gang":
				// Controllo Grado
				if !hasStaffRole(i.Member) {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: 4,
						Data: &discordgo.InteractionResponseData{
							Content: "❌ **Non hai l'autorizzazione di Staff per usare questo comando.**",
							Flags:   64,
						},
					})
					return
				}

				discordUser := optionMap["nome-discord"].UserValue(s)
				warnNum := optionMap["warn"].IntValue()
				motivo := optionMap["motivo"].StringValue()

				messaggio := fmt.Sprintf(
					"⚠️ **AMMONIMENTO (WARN) EMESSO**\n\n• **Nome Discord:** %s\n• **Warn:** %d/3\n• **Motivo:** %s\n• **Eseguito da:** %s",
					discordUser.Mention(), warnNum, motivo, i.Member.User.Mention(),
				)

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: 4,
					Data: &discordgo.InteractionResponseData{Content: messaggio},
				})

			case "blacklist-gang":
				// Controllo Grado
				if !hasStaffRole(i.Member) {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: 4,
						Data: &discordgo.InteractionResponseData{
							Content: "❌ **Non hai l'autorizzazione di Staff per usare questo comando.**",
							Flags:   64,
						},
					})
					return
				}

				discordUser := optionMap["nome-discord"].UserValue(s)
				robloxName := optionMap["nome-roblox"].StringValue()
				motivo := optionMap["motivo"].StringValue()

				messaggio := fmt.Sprintf(
					"🚫 **BLACKLIST REGISTRATA**\n\n• **Nome Discord:** %s\n• **Nome Roblox:** `%s`\n• **Motivo:** %s\n• **Eseguito da:** %s\n\n*Nota: L'ingresso in questo elenco preclude qualsiasi futuro arruolamento.*",
					discordUser.Mention(), robloxName, motivo, i.Member.User.Mention(),
				)

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: 4,
					Data: &discordgo.InteractionResponseData{Content: messaggio},
				})
			}
		}

		// Creazione Ticket/Canale Provino dal Menu a tendina
		if i.Type == discordgo.InteractionMessageComponent && i.MessageComponentData().CustomID == "ticket_gang" {
			utente := i.Member.User
			ch, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
				Name: "gang-" + utente.Username,
				Type: discordgo.ChannelTypeGuildText,
				PermissionOverwrites: []*discordgo.PermissionOverwrite{
					{ID: i.GuildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: 1024},
					{ID: utente.ID, Type: discordgo.PermissionOverwriteTypeMember, Allow: 3072},
					{ID: reclutatoriRole1, Type: discordgo.PermissionOverwriteTypeRole, Allow: 3072},
					{ID: reclutatoriRole2, Type: discordgo.PermissionOverwriteTypeRole, Allow: 3072},
					{ID: capoGangRole, Type: discordgo.PermissionOverwriteTypeRole, Allow: 3072},
				},
			})
			if err != nil { return }

			s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
				Content: "🚪 **RICHIESTA APERTA**\nCandidato: " + utente.Mention() + "\nIn attesa di: <@&" + reclutatoriRole1 + "> <@&" + reclutatoriRole2 + ">",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.Button{Label: "Accetta / Reclama ✋", Style: discordgo.PrimaryButton, CustomID: "btn_reclama_gang"},
						discordgo.Button{Label: "Rifiuta / Chiudi 🔒", Style: discordgo.DangerButton, CustomID: "btn_chiudi_gang"},
					}},
				},
			})

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: 4, Data: &discordgo.InteractionResponseData{Content: "✅ Richiesta inoltrata: <#"+ch.ID+">", Flags: 64},
			})
		}

		// Gestione dei Pulsanti
		if i.Type == discordgo.InteractionMessageComponent {
			if i.MessageComponentData().CustomID == "btn_reclama_gang" {
				s.ChannelEditComplex(i.ChannelID, &discordgo.ChannelEdit{
					PermissionOverwrites: []*discordgo.PermissionOverwrite{
						{ID: i.GuildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: 1024},
						{ID: i.Member.User.ID, Type: discordgo.PermissionOverwriteTypeMember, Allow: 3072},
						{ID: capoGangRole, Type: discordgo.PermissionOverwriteTypeRole, Allow: 3072},
					},
				})
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: 4, Data: &discordgo.InteractionResponseData{Content: "✅ Hai preso in carico questa richiesta!"},
				})
			}
			if i.MessageComponentData().CustomID == "btn_chiudi_gang" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: 4, Data: &discordgo.InteractionResponseData{Content: "🔒 Chiusura in corso..."},
				})
				time.Sleep(2 * time.Second)
				s.ChannelDelete(i.ChannelID)
			}
		}
	})

	s.Open()
	s.ApplicationCommandBulkOverwrite(s.State.User.ID, serverID, commands)
	
	fmt.Println("Bot Hamburg Gang Online! 🚀")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}
