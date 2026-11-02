package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "modernc.org/sqlite"
)

const guildID = "1488262125708116198"
const adminRoleID = "1488923224434802808" // Ruolo Amministrazione

var db *sql.DB
var mu sync.Mutex

var serverConfigs = make(map[string]*ServerConfig)

var defaultTaxes = GeneralTaxes{
	Conto:            100,
	Immatricolazione: 200,
	Immobili:         500,
	IntervalDays:     7,
	CustomDeadline:   "Ogni 7 giorni",
}

var stipendioFisso = 13000

// ---------------------------------------------------------------------------
// STRUTTURE DATI
// ---------------------------------------------------------------------------

type ServerConfig struct {
	PanelChannelID string
	EsitiChannelID string
	AuthRoleID     string
}

type GeneralTaxes struct {
	Conto            int
	Immatricolazione int
	Immobili         int
	IntervalDays     int
	CustomDeadline   string
}

type BankAccount struct {
	OwnerName   string
	RobloxName  string
	BankName    string
	Balance     int
	LastPayday  string
	LastTaxPaid string
}

// ---------------------------------------------------------------------------
// INIZIALIZZAZIONE DATABASE SQLITE (PERSISTENZA GLOBALE)
// ---------------------------------------------------------------------------

func initDatabase() {
	var err error
	db, err = sql.Open("sqlite", "./bot.db")
	if err != nil {
		log.Fatalf("❌ Errore apertura database SQLite: %v", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS bank_accounts (user_id TEXT PRIMARY KEY, owner_name TEXT, roblox_name TEXT, bank_name TEXT, balance INTEGER, last_payday TEXT, last_tax_paid TEXT);`,
		`CREATE TABLE IF NOT EXISTS warns (user_id TEXT PRIMARY KEY, count INTEGER, reason TEXT);`,
		`CREATE TABLE IF NOT EXISTS anagrafe (user_id TEXT PRIMARY KEY, roblox_user TEXT, nome_rp TEXT, cognome_rp TEXT, data_nascita TEXT);`,
		`CREATE TABLE IF NOT EXISTS porto_darmi (user_id TEXT PRIMARY KEY, roblox_user TEXT, nome_rp TEXT, motivazione TEXT);`,
		`CREATE TABLE IF NOT EXISTS immatricolazioni (targa TEXT PRIMARY KEY, user_id TEXT, roblox_user TEXT, modello_auto TEXT);`,
		`CREATE TABLE IF NOT EXISTS arresti (id INTEGER PRIMARY KEY AUTOINCREMENT, agente_id TEXT, arrestato_roblox TEXT, reati TEXT, pena TEXT, cauzione TEXT, data TEXT);`,
		`CREATE TABLE IF NOT EXISTS multe (id INTEGER PRIMARY KEY AUTOINCREMENT, agente_id TEXT, multato_roblox TEXT, importo TEXT, motivo TEXT, targa TEXT, data TEXT);`,
		`CREATE TABLE IF NOT EXISTS tax_overrides (user_id TEXT PRIMARY KEY, conto INTEGER, immatricolazione INTEGER, immobili INTEGER);`,
	}

	for _, q := range queries {
		if _, err = db.Exec(q); err != nil {
			log.Fatalf("❌ Errore creazione tabelle SQLite: %v", err)
		}
	}
	log.Println("✅ Database SQLite inizializzato e persistente!")
}

// ---------------------------------------------------------------------------
// BLOXLINK & AVATAR HELPER
// ---------------------------------------------------------------------------

func getRobloxAvatar(robloxUsername string) string {
	resp, err := http.Get(fmt.Sprintf("https://users.roblox.com/v1/users/search?keyword=%s&limit=1", robloxUsername))
	if err != nil {
		return "https://www.roblox.com/favicon.ico"
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			Id int `json:"id"`
		} `json:"data"`
	}
	if json.NewDecoder(resp.Body).Decode(&result) == nil && len(result.Data) > 0 {
		userId := result.Data[0].Id
		thumbResp, err := http.Get(fmt.Sprintf("https://thumbnails.roblox.com/v1/users/avatar-headshot?userIds=%d&size=420x420&format=Png&isCircular=false", userId))
		if err == nil {
			defer thumbResp.Body.Close()
			var thumbResult struct {
				Data []struct {
					ImageUrl string `json:"imageUrl"`
				} `json:"data"`
			}
			if json.NewDecoder(thumbResp.Body).Decode(&thumbResult) == nil && len(thumbResult.Data) > 0 {
				return thumbResult.Data[0].ImageUrl
			}
		}
	}
	return "https://www.roblox.com/favicon.ico"
}

// ---------------------------------------------------------------------------
// FUNZIONI DI SUPPORTO (HELPER)
// ---------------------------------------------------------------------------

func getOrCreateConfig(key string) *ServerConfig {
	if serverConfigs[key] == nil {
		serverConfigs[key] = &ServerConfig{}
	}
	return serverConfigs[key]
}

func parseEuro(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-3]) + "..."
}

func modalVal(d discordgo.ModalSubmitInteractionData, n int) string {
	if n >= len(d.Components) {
		return ""
	}
	row, ok := d.Components[n].(*discordgo.ActionsRow)
	if !ok || len(row.Components) == 0 {
		return ""
	}
	ti, ok := row.Components[0].(*discordgo.TextInput)
	if !ok {
		return ""
	}
	return ti.Value
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func inputRow(id, label string, style discordgo.TextInputStyle, value, placeholder string, required bool) discordgo.ActionsRow {
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.TextInput{
			CustomID:    id,
			Label:       label,
			Style:       style,
			Value:       value,
			Placeholder: placeholder,
			Required:    required,
		},
	}}
}

func openModal(s *discordgo.Session, i *discordgo.InteractionCreate, customID, title string, rows ...discordgo.ActionsRow) {
	comps := make([]discordgo.MessageComponent, 0, len(rows))
	for _, r := range rows {
		comps = append(comps, r)
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   customID,
			Title:      title,
			Components: comps,
		},
	})
}

func esitiChannel(key, fallback string) string {
	cfg := getOrCreateConfig(key)
	if cfg.EsitiChannelID != "" {
		return cfg.EsitiChannelID
	}
	return fallback
}

func sendApproval(s *discordgo.Session, channelID string, embed *discordgo.MessageEmbed, userID, yesLabel, noLabel string) {
	btnYes := discordgo.Button{Label: yesLabel, Style: discordgo.SuccessButton, CustomID: "approve_yes_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "✅"}}
	btnNo := discordgo.Button{Label: noLabel, Style: discordgo.DangerButton, CustomID: "approve_no_" + userID, Emoji: &discordgo.ComponentEmoji{Name: "❌"}}
	s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{btnYes, btnNo}},
		},
	})
}

func sendPanel(s *discordgo.Session, channelID string, embed *discordgo.MessageEmbed, btn *discordgo.Button) {
	msg := &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}}
	if btn != nil {
		msg.Components = []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{*btn}},
		}
	}
	s.ChannelMessageSendComplex(channelID, msg)
}

func hasAdminRole(member *discordgo.Member) bool {
	for _, r := range member.Roles {
		if r == adminRoleID {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// MAIN & SERVER HTTP (RENDER)
// ---------------------------------------------------------------------------

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

	initDatabase()

	token := strings.TrimSpace(os.Getenv("DISCORD_TOKEN"))
	token = strings.TrimPrefix(token, "Bot ")
	token = strings.TrimSpace(token)

	if token == "" {
		log.Fatalf("⚠️ ERRORE CRITICO: La variabile DISCORD_TOKEN non è impostata su Render!")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Errore creazione sessione Discord: %v", err)
	}

	dg.AddHandler(interactionHandler)

	if err = dg.Open(); err != nil {
		log.Fatalf("Errore apertura connessione Discord: %v", err)
	}
	defer dg.Close()

	log.Println("✅ Bot 'Rome Italy 2.0' avviato e connesso a Discord con successo!")
	registerCommands(dg)

	go startTaxScheduler(dg)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Arresto del bot in corso...")
}

func startTaxScheduler(s *discordgo.Session) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock()
		interval := time.Duration(defaultTaxes.IntervalDays) * 24 * time.Hour
		now := time.Now()

		rows, err := db.Query("SELECT user_id, owner_name, roblox_name, bank_name, balance, last_payday, last_tax_paid FROM bank_accounts")
		if err == nil {
			for rows.Next() {
				var userID string
				var acc BankAccount
				var lastTaxStr string
				if rows.Scan(&userID, &acc.OwnerName, &acc.RobloxName, &acc.BankName, &acc.Balance, &acc.LastPayday, &lastTaxStr) == nil {
					lastTax, _ := time.Parse(time.RFC3339, lastTaxStr)
					if lastTax.IsZero() {
						db.Exec("UPDATE bank_accounts SET last_tax_paid = ? WHERE user_id = ?", now.Format(time.RFC3339), userID)
						continue
					}

					if now.Sub(lastTax) >= interval {
						totale := defaultTaxes.Conto + defaultTaxes.Immatricolazione + defaultTaxes.Immobili
						if acc.Balance >= totale {
							acc.Balance -= totale
							db.Exec("UPDATE bank_accounts SET balance = ?, last_tax_paid = ? WHERE user_id = ?", acc.Balance, now.Format(time.RFC3339), userID)

							if ch, err := s.UserChannelCreate(userID); err == nil {
								s.ChannelMessageSend(ch.ID, fmt.Sprintf("🏛️ **AGENZIA DELLE ENTRATE:** Ti sono stati scalati **€ %d** di tasse.\n💰 Saldo residuo: € %d", totale, acc.Balance))
							}
						}
					}
				}
			}
			rows.Close()
		}
		mu.Unlock()
	}
}

// ---------------------------------------------------------------------------
// REGISTRAZIONE COMANDI SLASH
// ---------------------------------------------------------------------------

func registerCommands(s *discordgo.Session) {
	staffPerm := int64(32)

	setupSub := func(name, desc string) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        name,
			Description: desc,
		}
	}

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "conto",
			Description: "Visualizza il saldo del tuo conto corrente bancario",
		},
		{
			Name:        "bonifico",
			Description: "Invia un bonifico in denaro a un altro cittadino",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "destinatario", Description: "L'utente a cui inviare i soldi", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "importo", Description: "La cifra in euro da inviare", Required: true},
			},
		},
		{
			Name:        "aggiungi-soldi",
			Description: "Aggiunge soldi sul conto di un utente (Solo Amministrazione)",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "L'utente a cui accreditare i soldi", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "importo", Description: "La cifra in euro da aggiungere", Required: true},
			},
		},
		{
			Name:                     "warn",
			Description:              "Emetti un warn ufficiale a un cittadino",
			DefaultMemberPermissions: &staffPerm,
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "L'utente Discord da warnare", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "roblox", Description: "Nome utente Roblox", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivo del warn", Required: true},
			},
		},
		{
			Name:                     "ban-game",
			Description:              "Emetti un ban di gioco (Game Ban)",
			DefaultMemberPermissions: &staffPerm,
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "L'utente Discord da bannare", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "roblox", Description: "Nome utente Roblox", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "motivo", Description: "Motivo del ban", Required: true},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "giorni", Description: "Giorni di ban (0 per permaban)", Required: true},
			},
		},
		{
			Name:                     "database-polizia",
			Description:              "Cerca lo storico di arresti e multe per un utente Roblox",
			DefaultMemberPermissions: &staffPerm,
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "roblox", Description: "Username Roblox da cercare", Required: true},
			},
		},
		{
			Name:                     "fedina-penale",
			Description:              "Controlla la fedina penale completa di un utente",
			DefaultMemberPermissions: &staffPerm,
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "L'utente Discord da verificare", Required: true},
			},
		},
		{
			Name:                     "visualizza-cittadinanza",
			Description:              "Mostra la carta d'identità registrata di un utente",
			DefaultMemberPermissions: &staffPerm,
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "utente", Description: "L'utente Discord da controllare", Required: true},
			},
		},
		{
			Name:                     "backup",
			Description:              "Comandi di salvataggio e backup del Database SQLite (Solo Amministrazione)",
			DefaultMemberPermissions: &staffPerm,
			Options: []*discordgo.ApplicationCommandOption{
				setupSub("scarica", "Scarica il file bot.db originale di backup"),
				setupSub("dati", "Visualizza la panoramica di tutti i dati salvati su SQLite"),
			},
		},
		{
			Name:                     "visualizza",
			Description:              "Comandi di visualizzazione database per lo Staff",
			DefaultMemberPermissions: &staffPerm,
			Options: []*discordgo.ApplicationCommandOption{
				setupSub("anagrafe", "Visualizza tutte le carte d'identità registrate"),
				setupSub("porto-darmi", "Visualizza tutte le richieste di porto d'armi"),
				setupSub("immatricolazioni", "Visualizza tutti i veicoli immatricolati"),
				setupSub("warns", "Visualizza la lista di tutti i warn degli utenti"),
			},
		},
		{
			Name:                     "setup",
			Description:              "Configura i pannelli interattivi e i canali del server",
			DefaultMemberPermissions: &staffPerm,
			Options: []*discordgo.ApplicationCommandOption{
				setupSub("server", "Pannello generale per configurare canali e ruoli"),
				setupSub("apri-conto", "Invia il pannello bancario per l'apertura del conto"),
				setupSub("busta-paga", fmt.Sprintf("Invia il pannello per la busta paga (€ %d)", stipendioFisso)),
				setupSub("cittadinanza", "Invia il pannello anagrafe per la carta d'identità"),
				setupSub("porto-darmi", "Invia il pannello per la richiesta del porto d'armi"),
				setupSub("immatricolazione", "Invia il pannello PRA per immatricolare i veicoli"),
				setupSub("registra-arresto", "Invia il pannello per registrare arresti"),
				setupSub("emetti-multa", "Invia il pannello per le sanzioni stradali"),
				setupSub("tasse", "Gestisci e invia il pannello tasse"),
				setupSub("staff-panel", "Invia il pannello riservato alla moderazione"),
			},
		},
	}

	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildID, commands)
	if err != nil {
		log.Printf("Errore nella registrazione dei comandi: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GESTIONE INTERAZIONI
// ---------------------------------------------------------------------------

func interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	mu.Lock()
	defer mu.Unlock()

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleSlashCommands(s, i)
	case discordgo.InteractionMessageComponent:
		handleComponents(s, i)
	case discordgo.InteractionModalSubmit:
		handleModalSubmit(s, i)
	}
}

// ---------------------------------------------------------------------------
// ESECUZIONE COMANDI SLASH
// ---------------------------------------------------------------------------

func handleSlashCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	if i.Member == nil {
		respond(s, i, "❌ Questo comando funziona solo dentro il server.")
		return
	}

	if data.Name == "conto" {
		userID := i.Member.User.ID
		var acc BankAccount
		err := db.QueryRow("SELECT owner_name, roblox_name, bank_name, balance FROM bank_accounts WHERE user_id = ?", userID).Scan(&acc.OwnerName, &acc.RobloxName, &acc.BankName, &acc.Balance)
		if err != nil {
			respond(s, i, "❌ Non possiedi ancora un conto corrente registrato! Usa il pannello della banca per aprirne uno.")
			return
		}

		embed := &discordgo.MessageEmbed{
			Title: "🏛️ ESTRATTO CONTO BANCARIO",
			Color: 0x2ECC71,
			Description: fmt.Sprintf(
				"**Utente Discord:** <@%s>\n**Intestatario RP:** %s\n**Utente Roblox:** `@%s`\n**Banca:** %s\n💰 **Saldo Disponibile:** `€ %d`",
				userID, acc.OwnerName, acc.RobloxName, strings.ToUpper(acc.BankName), acc.Balance),
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	if data.Name == "bonifico" {
		senderID := i.Member.User.ID
		destUser := data.Options[0].UserValue(s)
		importo := int(data.Options[1].IntValue())

		if importo <= 0 {
			respond(s, i, "❌ L'importo del bonifico deve essere superiore a 0.")
			return
		}
		if senderID == destUser.ID {
			respond(s, i, "❌ Non puoi fare un bonifico a te stesso!")
			return
		}

		var senderBal int
		if db.QueryRow("SELECT balance FROM bank_accounts WHERE user_id = ?", senderID).Scan(&senderBal) != nil {
			respond(s, i, "❌ Non hai un conto corrente attivo!")
			return
		}
		var destBal int
		if db.QueryRow("SELECT balance FROM bank_accounts WHERE user_id = ?", destUser.ID).Scan(&destBal) != nil {
			respond(s, i, "❌ Il destinatario non possiede un conto corrente attivo.")
			return
		}

		if senderBal < importo {
			respond(s, i, fmt.Sprintf("⚠️ **Saldo insufficiente!** Hai `€ %d`, ma vuoi inviare `€ %d`.", senderBal, importo))
			return
		}

		db.Exec("UPDATE bank_accounts SET balance = balance - ? WHERE user_id = ?", importo, senderID)
		db.Exec("UPDATE bank_accounts SET balance = balance + ? WHERE user_id = ?", importo, destUser.ID)

		respond(s, i, fmt.Sprintf("✅ **Bonifico Effettuato!** Inviati **€ %d** a <@%s>.", importo, destUser.ID))
		if ch, err := s.UserChannelCreate(destUser.ID); err == nil {
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("💸 **BONIFICO RICEVUTO:** Hai ricevuto un bonifico di **€ %d** da <@%s>.", importo, senderID))
		}
		return
	}

	if data.Name == "aggiungi-soldi" {
		if !hasAdminRole(i.Member) {
			respond(s, i, "❌ Non hai i permessi necessari.")
			return
		}

		targetUser := data.Options[0].UserValue(s)
		importo := int(data.Options[1].IntValue())

		var dummy string
		if db.QueryRow("SELECT user_id FROM bank_accounts WHERE user_id = ?", targetUser.ID).Scan(&dummy) != nil {
			respond(s, i, "❌ L'utente specificato non possiede un conto corrente attivo.")
			return
		}

		db.Exec("UPDATE bank_accounts SET balance = balance + ? WHERE user_id = ?", importo, targetUser.ID)
		respond(s, i, fmt.Sprintf("✅ Aggiunti **€ %d** sul conto di <@%s>.", importo, targetUser.ID))
		if ch, err := s.UserChannelCreate(targetUser.ID); err == nil {
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("🏦 **ACCREDITO AMMINISTRAZIONE:** Ti sono stati accreditati **€ %d** sul conto.", importo))
		}
		return
	}

	if data.Name == "warn" {
		targetUser := data.Options[0].UserValue(s)
		robloxName := data.Options[1].StringValue()
		reason := data.Options[2].StringValue()

		var count int
		db.QueryRow("SELECT count FROM warns WHERE user_id = ?", targetUser.ID).Scan(&count)
		count++
		db.Exec("INSERT INTO warns (user_id, count, reason) VALUES (?, ?, ?) ON CONFLICT(user_id) DO UPDATE SET count = ?, reason = ?", targetUser.ID, count, reason, count, reason)

		avatarURL := getRobloxAvatar(robloxName)

		embed := &discordgo.MessageEmbed{
			Title:       "⚠️ NOTIFICA WARN UFFICIALE",
			Color:       0xF1C40F,
			Description: "Il cittadino ha ricevuto un richiamo ufficiale dallo Staff.",
			Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: avatarURL},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Nome Discord", Value: fmt.Sprintf("<@%s>", targetUser.ID), Inline: true},
				{Name: "Nome Roblox", Value: fmt.Sprintf("`@%s`", robloxName), Inline: true},
				{Name: "Motivo", Value: reason, Inline: false},
				{Name: "Warn Totali", Value: strconv.Itoa(count), Inline: true},
			},
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
		})

		if ch, err := s.UserChannelCreate(targetUser.ID); err == nil {
			s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}})
		}
		return
	}

	if data.Name == "ban-game" {
		targetUser := data.Options[0].UserValue(s)
		robloxName := data.Options[1].StringValue()
		reason := data.Options[2].StringValue()
		giorni := int(data.Options[3].IntValue())

		durataStr := fmt.Sprintf("%d giorni", giorni)
		if giorni == 0 {
			durataStr = "Permanente (Permaban)"
		}

		avatarURL := getRobloxAvatar(robloxName)

		embed := &discordgo.MessageEmbed{
			Title:       "🔨 GAME BAN - PROVVEDIMENTO DISCIPLINARE",
			Color:       0xE74C3C,
			Description: "Sei stato bannato dall'esperienza di gioco di Roblox.",
			Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: avatarURL},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Nome Discord", Value: fmt.Sprintf("<@%s>", targetUser.ID), Inline: true},
				{Name: "Nome Roblox", Value: fmt.Sprintf("`@%s`", robloxName), Inline: true},
				{Name: "Motivazione", Value: reason, Inline: false},
				{Name: "Durata", Value: durataStr, Inline: true},
			},
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
		})

		if ch, err := s.UserChannelCreate(targetUser.ID); err == nil {
			s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}})
		}
		return
	}

	if data.Name == "database-polizia" {
		robloxName := data.Options[0].StringValue()
		avatarURL := getRobloxAvatar(robloxName)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🚨 **BANCA DATI POLIZIA DI STATO / CARABINIERI**\n🔍 Storico per l'utente Roblox: `@%s`\n\n", robloxName))

		sb.WriteString("📌 **VERBALI D'ARRESTO:**\n")
		rowsA, errA := db.Query("SELECT agente_id, reati, pena, cauzione, data FROM arresti WHERE LOWER(arrestato_roblox) = LOWER(?)", robloxName)
		countArresti := 0
		if errA == nil {
			for rowsA.Next() {
				countArresti++
				var ag, reati, pena, cauz, dt string
				rowsA.Scan(&ag, &reati, &pena, &cauz, &dt)
				sb.WriteString(fmt.Sprintf("• [%s] Agente: <@%s> | **Reati:** %s | **Pena:** %s | **Cauzione:** €%s\n", dt, ag, reati, pena, cauz))
			}
			rowsA.Close()
		}
		if countArresti == 0 {
			sb.WriteString(" *Nessun arresto presente negli archivi.*\n")
		}

		sb.WriteString("\n📄 **SANZIONI STRADALI (MULTE):**\n")
		rowsM, errM := db.Query("SELECT agente_id, importo, motivo, targa, data FROM multe WHERE LOWER(multato_roblox) = LOWER(?)", robloxName)
		countMulte := 0
		if errM == nil {
			for rowsM.Next() {
				countMulte++
				var ag, imp, mot, tar, dt string
				rowsM.Scan(&ag, &imp, &mot, &tar, &dt)
				sb.WriteString(fmt.Sprintf("• [%s] Agente: <@%s> | **Importo:** €%s | **Motivo:** %s | **Targa:** %s\n", dt, ag, imp, mot, tar))
			}
			rowsM.Close()
		}
		if countMulte == 0 {
			sb.WriteString(" *Nessuna multa presente negli archivi.*\n")
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🚨 ARCHIVIO GIUDIZIARIO POLIZIA",
			Color:       0x34495E,
			Description: sb.String(),
			Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: avatarURL},
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
		})
		return
	}

	if data.Name == "fedina-penale" {
		targetUser := data.Options[0].UserValue(s)

		var robloxName string
		db.QueryRow("SELECT roblox_user FROM anagrafe WHERE user_id = ?", targetUser.ID).Scan(&robloxName)
		if robloxName == "" {
			db.QueryRow("SELECT roblox_name FROM bank_accounts WHERE user_id = ?", targetUser.ID).Scan(&robloxName)
		}

		var warnCount int
		var warnReason string
		db.QueryRow("SELECT count, reason FROM warns WHERE user_id = ?", targetUser.ID).Scan(&warnCount, &warnReason)

		var countArresti int
		db.QueryRow("SELECT COUNT(*) FROM arresti WHERE agente_id = ? OR LOWER(arrestato_roblox) = LOWER(?)", targetUser.ID, robloxName).Scan(&countArresti)

		var countMulte int
		db.QueryRow("SELECT COUNT(*) FROM multe WHERE agente_id = ? OR LOWER(multato_roblox) = LOWER(?)", targetUser.ID, robloxName).Scan(&countMulte)

		stato := "🟢 PULITA"
		color := 0x2ECC71
		if countArresti > 0 || warnCount >= 3 {
			stato = "🔴 PREGIUDICATO / SPORCA"
			color = 0xE74C3C
		} else if countMulte > 0 || warnCount > 0 {
			stato = "🟡 SEGNALATO"
			color = 0xF1C40F
		}

		avatarURL := "https://www.roblox.com/favicon.ico"
		if robloxName != "" {
			avatarURL = getRobloxAvatar(robloxName)
		}

		embed := &discordgo.MessageEmbed{
			Title:     "📜 ESTRATTO FEDINA PENALE",
			Color:     color,
			Thumbnail: &discordgo.MessageEmbedThumbnail{URL: avatarURL},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Cittadino", Value: fmt.Sprintf("<@%s>", targetUser.ID), Inline: true},
				{Name: "Roblox User", Value: fmt.Sprintf("`@%s`", robloxName), Inline: true},
				{Name: "Stato Giudiziario", Value: stato, Inline: true},
				{Name: "Arresti Subiti", Value: strconv.Itoa(countArresti), Inline: true},
				{Name: "Multe Ricevute", Value: strconv.Itoa(countMulte), Inline: true},
				{Name: "Warn Accumulati", Value: strconv.Itoa(warnCount), Inline: true},
			},
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
		})
		return
	}

	if data.Name == "visualizza-cittadinanza" {
		targetUser := data.Options[0].UserValue(s)

		var rec struct {
			RobloxUser  string
			NomeRP      string
			CognomeRP   string
			DataNascita string
		}

		err := db.QueryRow("SELECT roblox_user, nome_rp, cognome_rp, data_nascita FROM anagrafe WHERE user_id = ?", targetUser.ID).Scan(&rec.RobloxUser, &rec.NomeRP, &rec.CognomeRP, &rec.DataNascita)
		if err != nil {
			respond(s, i, "❌ Questo utente non possiede ancora una Carta d'Identità registrata nell'Anagrafe.")
			return
		}

		avatarURL := getRobloxAvatar(rec.RobloxUser)

		embed := &discordgo.MessageEmbed{
			Title:     "🇮🇹 CARTA D'IDENTITÀ UFFICIALE - ROME ITALY 2.0",
			Color:     0x3498DB,
			Thumbnail: &discordgo.MessageEmbedThumbnail{URL: avatarURL},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Nome RP", Value: rec.NomeRP, Inline: true},
				{Name: "Cognome RP", Value: rec.CognomeRP, Inline: true},
				{Name: "Data di Nascita", Value: rec.DataNascita, Inline: true},
				{Name: "Account Discord", Value: fmt.Sprintf("<@%s>", targetUser.ID), Inline: true},
				{Name: "Account Roblox", Value: fmt.Sprintf("`@%s`", rec.RobloxUser), Inline: true},
			},
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
		})
		return
	}

	if len(data.Options) == 0 {
		return
	}
	sub := data.Options[0].Name

	// ---------------------------------------------------------------------------
	// NUOVO COMANDO BACKUP SQLITE
	// ---------------------------------------------------------------------------

	if data.Name == "backup" {
		if !hasAdminRole(i.Member) {
			respond(s, i, "❌ Non hai i permessi di Amministrazione per eseguire questo comando.")
			return
		}

		if sub == "scarica" {
			file, err := os.Open("./bot.db")
			if err != nil {
				respond(s, i, "❌ Errore nell'apertura del file database: bot.db non trovato.")
				return
			}
			defer file.Close()

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "💾 **BACKUP DATABASE COMPLETO (bot.db)**\nEcco il file del database SQLite contenente tutti i dati registrati:",
					Flags:   discordgo.MessageFlagsEphemeral,
					Files: []*discordgo.File{
						{
							Name:        "bot.db",
							ContentType: "application/x-sqlite3",
							Reader:      file,
						},
					},
				},
			})
			return
		}

		if sub == "dati" {
			var countBanche, countWarns, countAnagrafe, countPorto, countImm, countArresti, countMulte, countTax int

			db.QueryRow("SELECT COUNT(*) FROM bank_accounts").Scan(&countBanche)
			db.QueryRow("SELECT COUNT(*) FROM warns").Scan(&countWarns)
			db.QueryRow("SELECT COUNT(*) FROM anagrafe").Scan(&countAnagrafe)
			db.QueryRow("SELECT COUNT(*) FROM porto_darmi").Scan(&countPorto)
			db.QueryRow("SELECT COUNT(*) FROM immatricolazioni").Scan(&countImm)
			db.QueryRow("SELECT COUNT(*) FROM arresti").Scan(&countArresti)
			db.QueryRow("SELECT COUNT(*) FROM multe").Scan(&countMulte)
			db.QueryRow("SELECT COUNT(*) FROM tax_overrides").Scan(&countTax)

			embed := &discordgo.MessageEmbed{
				Title: "📊 PANORAMICA DATI SALVATI SU SQLITE (bot.db)",
				Color: 0x9B59B6,
				Description: fmt.Sprintf(
					"**Stato Database:** 🟢 Connesso e Operativo\n\n"+
						"💳 **Conti Bancari Registrati:** `%d`\n"+
						"🇮🇹 **Carte d'Identità Anagrafe:** `%d`\n"+
						"🚨 **Verbali d'Arresto Archiviati:** `%d`\n"+
						"📄 **Multe Stradali Registrate:** `%d`\n"+
						"🚗 **Veicoli Immatricolati (PRA):** `%d`\n"+
						"🔫 **Licenze Porto d'Armi:** `%d`\n"+
						"⚠️ **Cittadini Warnati:** `%d`\n"+
						"👤 **Tasse Personalizzate:** `%d`",
					countBanche, countAnagrafe, countArresti, countMulte, countImm, countPorto, countWarns, countTax),
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
	}

	if data.Name == "visualizza" {
		var sb strings.Builder
		switch sub {
		case "anagrafe":
			sb.WriteString("📋 **LISTA CITTADINANZA / ANAGRAFE:**\n\n")
			rows, _ := db.Query("SELECT user_id, roblox_user, nome_rp, cognome_rp, data_nascita FROM anagrafe")
			if rows != nil {
				for rows.Next() {
					var uid, rob, nom, cog, nas string
					rows.Scan(&uid, &rob, &nom, &cog, &nas)
					sb.WriteString(fmt.Sprintf("• <@%s> | **RP:** %s %s | **Roblox:** `@%s` | **Nato:** %s\n", uid, nom, cog, rob, nas))
				}
				rows.Close()
			}
		case "porto-darmi":
			sb.WriteString("🔫 **LISTA PORTO D'ARMI:**\n\n")
			rows, _ := db.Query("SELECT user_id, roblox_user, nome_rp, motivazione FROM porto_darmi")
			if rows != nil {
				for rows.Next() {
					var uid, rob, nom, mot string
					rows.Scan(&uid, &rob, &nom, &mot)
					sb.WriteString(fmt.Sprintf("• <@%s> | **RP:** %s | **Roblox:** `@%s` | **Motivazione:** %s\n", uid, nom, rob, mot))
				}
				rows.Close()
			}
		case "immatricolazioni":
			sb.WriteString("🚗 **LISTA VEICOLI (PRA):**\n\n")
			rows, _ := db.Query("SELECT targa, user_id, roblox_user, modello_auto FROM immatricolazioni")
			if rows != nil {
				for rows.Next() {
					var tar, uid, rob, mod string
					rows.Scan(&tar, &uid, &rob, &mod)
					sb.WriteString(fmt.Sprintf("• <@%s> | **Veicolo:** %s | **Targa:** `%s` | **Roblox:** `@%s`\n", uid, mod, tar, rob))
				}
				rows.Close()
			}
		case "warns":
			sb.WriteString("⚠️ **LISTA WARN UTENTI:**\n\n")
			rows, _ := db.Query("SELECT user_id, count, reason FROM warns")
			if rows != nil {
				for rows.Next() {
					var uid, reason string
					var count int
					rows.Scan(&uid, &count, &reason)
					sb.WriteString(fmt.Sprintf("• <@%s> | **Warn:** `%d` | **Motivo:** %s\n", uid, count, reason))
				}
				rows.Close()
			}
		}
		respond(s, i, truncate(sb.String(), 1900))
		return
	}

	if data.Name == "setup" {
		switch sub {
		case "server":
			sendServerSetupPanel(s, i)
		case "tasse":
			sendTasseSetupPanel(s, i)
		default:
			sendPanelPreview(s, i, sub)
		}
	}
}

// ---------------------------------------------------------------------------
// PANNELLI DI CONFIGURAZIONE E INTERAZIONI BOTTONI
// ---------------------------------------------------------------------------

func channelInfoText(key string) string {
	cfg := getOrCreateConfig(key)
	if cfg.PanelChannelID == "" {
		return "⚠️ *Nessun canale impostato in `/setup server`.*"
	}
	return fmt.Sprintf("📍 *Canale:* <#%s>", cfg.PanelChannelID)
}

func sendTasseSetupPanel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var customCount int
	db.QueryRow("SELECT COUNT(*) FROM tax_overrides").Scan(&customCount)

	embed := &discordgo.MessageEmbed{
		Title: "🏛️ GESTIONE FISCALE E TASSE - ROME ITALY 2.0",
		Description: fmt.Sprintf(
			"**Tasse Generali Standard:**\n"+
				"• Tassa Conto: € %d\n"+
				"• Tassa Immatricolazione: € %d\n"+
				"• Tassa Immobili: € %d\n"+
				"• Frequenza Prelievo: ogni %d giorni (%s)\n\n"+
				"👤 **Utenti con Tasse Personalizzate:** `%d`\n\n%s",
			defaultTaxes.Conto, defaultTaxes.Immatricolazione, defaultTaxes.Immobili,
			defaultTaxes.IntervalDays, defaultTaxes.CustomDeadline,
			customCount, channelInfoText("tasse")),
		Color: 0x95A5A6,
	}

	row1 := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Label: "Invia Pannello", Style: discordgo.SuccessButton, CustomID: "send_panel_to_channel_tasse", Emoji: &discordgo.ComponentEmoji{Name: "📨"}},
		discordgo.Button{Label: "Tasse Generali", Style: discordgo.PrimaryButton, CustomID: "btn_config_tasse_generali", Emoji: &discordgo.ComponentEmoji{Name: "⚙️"}},
		discordgo.Button{Label: "Frequenza", Style: discordgo.PrimaryButton, CustomID: "btn_config_tasse_time", Emoji: &discordgo.ComponentEmoji{Name: "⏱️"}},
	}}

	row2 := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Label: "Tassa Personalizzata", Style: discordgo.SecondaryButton, CustomID: "btn_config_tasse_user", Emoji: &discordgo.ComponentEmoji{Name: "👤"}},
		discordgo.Button{Label: "Rimuovi Personalizzata", Style: discordgo.DangerButton, CustomID: "btn_reset_tasse_user", Emoji: &discordgo.ComponentEmoji{Name: "🗑️"}},
	}}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{row1, row2},
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func sendPanelPreview(s *discordgo.Session, i *discordgo.InteractionCreate, serviceKey string) {
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📤 INVIA PANNELLO: %s", strings.ToUpper(serviceKey)),
		Description: fmt.Sprintf("Clicca sul pulsante per pubblicare il pannello.\n\n%s", channelInfoText(serviceKey)),
		Color:       0x3498DB,
	}
	btnSend := discordgo.Button{
		Label:    "Invia Pannello",
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
		Title:       "⚙️ CONFIGURAZIONE GLOBALE COMANDI",
		Description: "Seleziona dal menu il comando da configurare:",
		Color:       0x0099FF,
	}

	opt := func(label, value, emoji string) discordgo.SelectMenuOption {
		return discordgo.SelectMenuOption{
			Label:       label,
			Value:       value,
			Description: "Imposta canali e ruoli",
			Emoji:       &discordgo.ComponentEmoji{Name: emoji},
		}
	}

	selectMenu := discordgo.SelectMenu{
		CustomID:    "setup_server_select",
		Placeholder: "Scegli il comando...",
		Options: []discordgo.SelectMenuOption{
			opt("Cittadinanza", "cittadinanza", "🇮🇹"),
			opt("Porto d'Armi", "porto-darmi", "🔫"),
			opt("Apertura Conto", "apri-conto", "💳"),
			opt("Busta Paga", "busta-paga", "💵"),
			opt("Immatricolazione", "immatricolazione", "🚗"),
			opt("Registro Arresti", "registra-arresto", "🚨"),
			opt("Emetti Multa", "emetti-multa", "📄"),
			opt("Tasse", "tasse", "🏛️"),
			opt("Staff Panel", "staff-panel", "🛡️"),
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
	values := i.MessageComponentData().Values

	if i.Member == nil {
		return
	}

	if strings.HasPrefix(customID, "send_panel_to_channel_") {
		serviceKey := strings.TrimPrefix(customID, "send_panel_to_channel_")
		targetChannel := getOrCreateConfig(serviceKey).PanelChannelID
		if targetChannel == "" {
			targetChannel = i.ChannelID
		}
		publishPublicPanel(s, targetChannel, serviceKey)
		respond(s, i, fmt.Sprintf("✅ **Pannello inviato nel canale <#%s>!**", targetChannel))
		return
	}

	switch customID {
	case "btn_config_tasse_generali":
		openModal(s, i, "modal_config_tasse_base", "Configura Tasse Generali",
			inputRow("input_tax_conto", "Tassa Conto (€)", discordgo.TextInputShort, strconv.Itoa(defaultTaxes.Conto), "", true),
			inputRow("input_tax_imm", "Tassa Immatricolazione (€)", discordgo.TextInputShort, strconv.Itoa(defaultTaxes.Immatricolazione), "", true),
			inputRow("input_tax_immobili", "Tassa Immobili (€)", discordgo.TextInputShort, strconv.Itoa(defaultTaxes.Immobili), "", true),
		)
		return

	case "btn_config_tasse_time":
		openModal(s, i, "modal_config_tasse_time", "Tempo e Scadenza Tasse",
			inputRow("input_tax_days", "Ogni quanti giorni pagare?", discordgo.TextInputShort, strconv.Itoa(defaultTaxes.IntervalDays), "Es. 7", true),
			inputRow("input_tax_deadline_note", "Note/Scadenza", discordgo.TextInputShort, defaultTaxes.CustomDeadline, "Es. Ogni settimana", true),
		)
		return

	case "btn_config_tasse_user", "btn_reset_tasse_user":
		prefix, text := "seluser_tasse", "👤 Scegli l'utente per personalizzare le tasse:"
		if customID == "btn_reset_tasse_user" {
			prefix, text = "seluserreset_tasse", "🗑️ Scegli l'utente a cui ripristinare le tasse generali:"
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: text,
				Flags:   discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{CustomID: prefix, Placeholder: "Seleziona utente...", MenuType: discordgo.UserSelectMenu},
					}},
				},
			},
		})
		return

	case "seluser_tasse":
		if len(values) == 0 {
			return
		}
		target := values[0]
		var cConto, cImm, cImmobili int
		err := db.QueryRow("SELECT conto, immatricolazione, immobili FROM tax_overrides WHERE user_id = ?", target).Scan(&cConto, &cImm, &cImmobili)
		if err != nil {
			cConto, cImm, cImmobili = defaultTaxes.Conto, defaultTaxes.Immatricolazione, defaultTaxes.Immobili
		}
		openModal(s, i, "modal_tasse_user_"+target, "Tasse Personalizzate Utente",
			inputRow("u_conto", "Tassa Conto (€)", discordgo.TextInputShort, strconv.Itoa(cConto), "", true),
			inputRow("u_imm", "Tassa Immatricolazione (€)", discordgo.TextInputShort, strconv.Itoa(cImm), "", true),
			inputRow("u_immobili", "Tassa Immobili (€)", discordgo.TextInputShort, strconv.Itoa(cImmobili), "", true),
		)
		return

	case "seluserreset_tasse":
		if len(values) == 0 {
			return
		}
		target := values[0]
		db.Exec("DELETE FROM tax_overrides WHERE user_id = ?", target)
		respond(s, i, fmt.Sprintf("✅ Tassa personalizzata rimossa con successo per <@%s>.", target))
		return
	}

	if strings.HasPrefix(customID, "approve_yes_") || strings.HasPrefix(customID, "approve_no_") {
		parts := strings.Split(customID, "_")
		if len(parts) < 3 || len(i.Message.Embeds) == 0 {
			return
		}
		azione := parts[1]
		targetUserID := parts[2]

		newEmbed := *i.Message.Embeds[0]
		if azione == "yes" {
			newEmbed.Color = 0x2ECC71
			newEmbed.Title = "✅ APPROVATO DALLO STAFF"
			newEmbed.Description += fmt.Sprintf("\n\n**Stato:** Approvato da <@%s>", i.Member.User.ID)
		} else {
			newEmbed.Color = 0xE74C3C
			newEmbed.Title = "❌ RIFIUTATO DALLO STAFF"
			newEmbed.Description += fmt.Sprintf("\n\n**Stato:** Rifiutato da <@%s>", i.Member.User.ID)
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{&newEmbed}, Components: []discordgo.MessageComponent{}},
		})

		if ch, err := s.UserChannelCreate(targetUserID); err == nil {
			if azione == "yes" {
				s.ChannelMessageSend(ch.ID, "🎉 **Esito Modulo:** La tua richiesta nel server **Rome Italy 2.0** è stata **APPROVATA** dallo Staff!")
			} else {
				s.ChannelMessageSend(ch.ID, "❌ **Esito Modulo:** La tua richiesta nel server **Rome Italy 2.0** è stata **RIFIUTATA** dallo Staff.")
			}
		}
		return
	}

	if customID == "setup_server_select" {
		selected := "cittadinanza"
		if len(values) > 0 {
			selected = values[0]
		}

		chanTypes := []discordgo.ChannelType{discordgo.ChannelTypeGuildText}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("🛠️ **Configurazione per: %s**", strings.ToUpper(selected)),
				Flags:   discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{CustomID: "selchanpanel_" + selected, Placeholder: "Canale pannello (#)...", MenuType: discordgo.ChannelSelectMenu, ChannelTypes: chanTypes},
					}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{CustomID: "selchanesiti_" + selected, Placeholder: "Canale esiti (#)...", MenuType: discordgo.ChannelSelectMenu, ChannelTypes: chanTypes},
					}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{CustomID: "selroleauth_" + selected, Placeholder: "Ruolo autorizzato (@)...", MenuType: discordgo.RoleSelectMenu},
					}},
				},
			},
		})
		return
	}

	if strings.HasPrefix(customID, "selchanpanel_") || strings.HasPrefix(customID, "selchanesiti_") || strings.HasPrefix(customID, "selroleauth_") {
		if len(values) > 0 {
			v := values[0]
			switch {
			case strings.HasPrefix(customID, "selchanpanel_"):
				getOrCreateConfig(strings.TrimPrefix(customID, "selchanpanel_")).PanelChannelID = v
			case strings.HasPrefix(customID, "selchanesiti_"):
				getOrCreateConfig(strings.TrimPrefix(customID, "selchanesiti_")).EsitiChannelID = v
			case strings.HasPrefix(customID, "selroleauth_"):
				getOrCreateConfig(strings.TrimPrefix(customID, "selroleauth_")).AuthRoleID = v
			}
		}
		respond(s, i, "✅ **Salvato con successo!**")
		return
	}

	switch customID {
	case "btn_paga_tasse":
		userID := i.Member.User.ID
		var balance int
		err := db.QueryRow("SELECT balance FROM bank_accounts WHERE user_id = ?", userID).Scan(&balance)
		if err != nil {
			respond(s, i, "❌ Devi prima aprire un conto corrente!")
			return
		}

		totale := defaultTaxes.Conto + defaultTaxes.Immatricolazione + defaultTaxes.Immobili
		if balance < totale {
			respond(s, i, fmt.Sprintf("⚠️ **Saldo insufficiente!** Tasse: **€ %d**, saldo: `€ %d`.", totale, balance))
			return
		}

		db.Exec("UPDATE bank_accounts SET balance = balance - ?, last_tax_paid = ? WHERE user_id = ?", totale, time.Now().Format(time.RFC3339), userID)
		respond(s, i, fmt.Sprintf("🏛️ **Tasse Versate!** Totale: `€ %d`", totale))

		embedEsito := &discordgo.MessageEmbed{
			Title: "🏛️ RICEVUTA PAGAMENTO TASSE STATALI",
			Color: 0x2ECC71,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Cittadino", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
				{Name: "Totale", Value: fmt.Sprintf("`€ %d`", totale), Inline: true},
			},
		}
		s.ChannelMessageSendComplex(esitiChannel("tasse", i.ChannelID), &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embedEsito}})

	case "btn_apri_conto":
		var dummy string
		if db.QueryRow("SELECT user_id FROM bank_accounts WHERE user_id = ?", i.Member.User.ID).Scan(&dummy) == nil {
			respond(s, i, "❌ Hai già un conto corrente aperto!")
			return
		}
		selectBanche := discordgo.SelectMenu{
			CustomID:    "select_banca_choice",
			Placeholder: "Seleziona la banca...",
			Options: []discordgo.SelectMenuOption{
				{Label: "Banca Sella", Value: "banca_sella", Emoji: &discordgo.ComponentEmoji{Name: "🏛️"}},
				{Label: "Poste Italiane", Value: "poste", Emoji: &discordgo.ComponentEmoji{Name: "📮"}},
				{Label: "Intesa Sanpaolo", Value: "intesa", Emoji: &discordgo.ComponentEmoji{Name: "🏦"}},
				{Label: "UniCredit", Value: "unicredit", Emoji: &discordgo.ComponentEmoji{Name: "🔴"}},
				{Label: "Revolut", Value: "revolut", Emoji: &discordgo.ComponentEmoji{Name: "💳"}},
			},
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Scegli la banca (riceverai **€ 7.000** di bonus):",
				Flags:   discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{selectBanche}},
				},
			},
		})

	case "select_banca_choice":
		selected := "banca_generica"
		if len(values) > 0 {
			selected = values[0]
		}
		openModal(s, i, "modal_conferma_conto_"+selected, "Apertura Conto Corrente",
			inputRow("input_nome_cognome_rp", "Nome e Cognome RP", discordgo.TextInputShort, "", "Es. Mario Rossi", true),
			inputRow("input_roblox_username", "Username Roblox", discordgo.TextInputShort, "", "Es. RobloxUser123", true),
		)

	case "btn_richiedi_cittadinanza":
		openModal(s, i, "modal_cittadinanza", "Registrazione Carta d'Identità",
			inputRow("input_nome_rp", "Nome RP", discordgo.TextInputShort, "", "Es. Mario", true),
			inputRow("input_cognome_rp", "Cognome RP", discordgo.TextInputShort, "", "Es. Rossi", true),
			inputRow("input_roblox_user", "Username Roblox", discordgo.TextInputShort, "", "Es. RobloxPlayer1", true),
			inputRow("input_data_nascita", "Data di Nascita RP", discordgo.TextInputShort, "", "GG/MM/AAAA", true),
		)

	case "btn_richiedi_porto_darmi":
		openModal(s, i, "modal_porto_darmi", "Richiesta Licenza Porto d'Armi",
			inputRow("input_nome_cognome_pda", "Nome e Cognome RP", discordgo.TextInputShort, "", "", true),
			inputRow("input_roblox_pda", "Username Roblox", discordgo.TextInputShort, "", "", true),
			inputRow("input_motivazione_pda", "Motivazione", discordgo.TextInputParagraph, "", "", true),
		)

	case "btn_immatricolazione":
		openModal(s, i, "modal_immatricolazione", "Immatricolazione Veicolo (PRA)",
			inputRow("input_roblox_imm", "Username Roblox", discordgo.TextInputShort, "", "", true),
			inputRow("input_modello_auto", "Modello Veicolo", discordgo.TextInputShort, "", "", true),
			inputRow("input_targa", "Targa Desiderata", discordgo.TextInputShort, "", "", true),
		)

	case "btn_registra_arresto":
		openModal(s, i, "modal_registra_arresto", "Verbale d'Arresto",
			inputRow("input_utente_arrestato", "Utente Arrestato (Roblox)", discordgo.TextInputShort, "", "", true),
			inputRow("input_reati", "Reati", discordgo.TextInputParagraph, "", "", true),
			inputRow("input_prigione", "Pena", discordgo.TextInputShort, "", "", true),
			inputRow("input_cauzione", "Cauzione", discordgo.TextInputShort, "", "5000", false),
		)

	case "btn_emetti_multa":
		openModal(s, i, "modal_emetti_multa", "Sanzione Stradale",
			inputRow("input_utente_multato", "Utente Multato (Roblox)", discordgo.TextInputShort, "", "", true),
			inputRow("input_importo_multa", "Importo (€)", discordgo.TextInputShort, "", "150", true),
			inputRow("input_motivo_multa", "Motivazione", discordgo.TextInputParagraph, "", "", true),
			inputRow("input_targa_multa", "Targa", discordgo.TextInputShort, "", "", false),
		)

	case "btn_riscuoti_stipendio":
		userID := i.Member.User.ID
		var dummy string
		if db.QueryRow("SELECT user_id FROM bank_accounts WHERE user_id = ?", userID).Scan(&dummy) != nil {
			respond(s, i, "❌ Devi prima aprire un conto corrente!")
			return
		}
		db.Exec("UPDATE bank_accounts SET balance = balance + ? WHERE user_id = ?", stipendioFisso, userID)
		respond(s, i, fmt.Sprintf("✅ **Busta Paga Riscossa!** +€ %d", stipendioFisso))
	}
}

func publishPublicPanel(s *discordgo.Session, channelID string, serviceKey string) {
	switch serviceKey {
	case "apri-conto":
		sendPanel(s, channelID, &discordgo.MessageEmbed{
			Title:       "🏛️ APERTURA CONTO CORRENTE - ROME ITALY 2.0",
			Description: "Clicca per scegliere la banca e ricevere **€ 7.000** di bonus!",
			Color:       0x2ECC71,
		}, &discordgo.Button{Label: "Apri Conto Corrente", Style: discordgo.SuccessButton, CustomID: "btn_apri_conto", Emoji: &discordgo.ComponentEmoji{Name: "💳"}})
	case "porto-darmi":
		sendPanel(s, channelID, &discordgo.MessageEmbed{
			Title:       "🔫 RICHIESTA PORTO D'ARMI - ROME ITALY 2.0",
			Description: "Invia la richiesta di licenza porto d'armi.",
			Color:       0xE67E22,
		}, &discordgo.Button{Label: "Richiedi Porto d'Armi", Style: discordgo.PrimaryButton, CustomID: "btn_richiedi_porto_darmi", Emoji: &discordgo.ComponentEmoji{Name: "📜"}})
	case "busta-paga":
		sendPanel(s, channelID, &discordgo.MessageEmbed{
			Title:       "💼 RISCOSSIONE BUSTA PAGA - ROME ITALY 2.0",
			Description: fmt.Sprintf("Riscuoti la busta paga settimanale di **€ %d**!", stipendioFisso),
			Color:       0xF1C40F,
		}, &discordgo.Button{Label: fmt.Sprintf("Riscuoti Busta Paga (€ %d)", stipendioFisso), Style: discordgo.PrimaryButton, CustomID: "btn_riscuoti_stipendio", Emoji: &discordgo.ComponentEmoji{Name: "💵"}})
	case "cittadinanza":
		sendPanel(s, channelID, &discordgo.MessageEmbed{
			Title:       "🇮🇹 ANAGRAFE CENTRALE - ROME ITALY 2.0",
			Description: "Richiedi la tua Carta d'Identità e la Cittadinanza ufficiali.",
			Color:       0x3498DB,
		}, &discordgo.Button{Label: "Richiedi Cittadinanza", Style: discordgo.PrimaryButton, CustomID: "btn_richiedi_cittadinanza", Emoji: &discordgo.ComponentEmoji{Name: "🆔"}})
	case "immatricolazione":
		sendPanel(s, channelID, &discordgo.MessageEmbed{
			Title:       "🚗 UFFICIO PRA - IMMATRICOLAZIONE VEICOLI",
			Description: "Immatricola e registra il tuo veicolo nel server.",
			Color:       0x9B59B6,
		}, &discordgo.Button{Label: "Immatricola Veicolo", Style: discordgo.PrimaryButton, CustomID: "btn_immatricolazione", Emoji: &discordgo.ComponentEmoji{Name: "🚘"}})
	case "registra-arresto":
		sendPanel(s, channelID, &discordgo.MessageEmbed{
			Title:       "🚨 BANCA DATI POLIZIA DI STATO / CARABINIERI",
			Description: "Compila il verbale dei fermi ed arresti.",
			Color:       0xE74C3C,
		}, &discordgo.Button{Label: "Registra Arresto", Style: discordgo.DangerButton, CustomID: "btn_registra_arresto", Emoji: &discordgo.ComponentEmoji{Name: "📝"}})
	case "emetti-multa":
		sendPanel(s, channelID, &discordgo.MessageEmbed{
			Title:       "📄 POLIZIA LOCALE - EMISSIONE VERBALE MULTA",
			Description: "Emetti una sanzione stradale.",
			Color:       0xE74C3C,
		}, &discordgo.Button{Label: "Emetti Multa", Style: discordgo.DangerButton, CustomID: "btn_emetti_multa", Emoji: &discordgo.ComponentEmoji{Name: "📄"}})
	case "tasse":
		sendPanel(s, channelID, &discordgo.MessageEmbed{
			Title:       "🏛️ AGENZIA DELLE ENTRATE - PAGAMENTO TASSE STATALI",
			Description: fmt.Sprintf("Versa le tue imposte.\n📌 **Frequenza:** %s", defaultTaxes.CustomDeadline),
			Color:       0x95A5A6,
		}, &discordgo.Button{Label: "Paga Tasse Statali", Style: discordgo.SuccessButton, CustomID: "btn_paga_tasse", Emoji: &discordgo.ComponentEmoji{Name: "💸"}})
	case "staff-panel":
		sendPanel(s, channelID, &discordgo.MessageEmbed{
			Title:       "🛡️ PANNELLO STAFF - ROME ITALY 2.0",
			Description: "Comandi:\n• `/warn`\n• `/ban-game`\n• `/database-polizia`\n• `/fedina-penale`\n• `/visualizza-cittadinanza`\n• `/backup`\n• `/bonifico`\n• `/aggiungi-soldi` (Admin)\n• `/visualizza`",
			Color:       0x2C3E50,
		}, nil)
	}
}

func handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	customID := data.CustomID

	if i.Member == nil {
		return
	}
	userID := i.Member.User.ID
	nowDate := time.Now().Format("02/01/2006 15:04")

	if customID == "modal_config_tasse_base" {
		defaultTaxes.Conto = parseEuro(modalVal(data, 0))
		defaultTaxes.Immatricolazione = parseEuro(modalVal(data, 1))
		defaultTaxes.Immobili = parseEuro(modalVal(data, 2))
		respond(s, i, "✅ **Tasse Generali Aggiornate!**")
		return
	}

	if customID == "modal_config_tasse_time" {
		days := parseEuro(modalVal(data, 0))
		if days < 1 {
			days = 1
		}
		defaultTaxes.IntervalDays = days
		defaultTaxes.CustomDeadline = modalVal(data, 1)
		respond(s, i, "✅ **Tempo Tasse Aggiornato!**")
		return
	}

	if strings.HasPrefix(customID, "modal_tasse_user_") {
		target := strings.TrimPrefix(customID, "modal_tasse_user_")
		cConto := parseEuro(modalVal(data, 0))
		cImm := parseEuro(modalVal(data, 1))
		cImmobili := parseEuro(modalVal(data, 2))

		db.Exec("INSERT OR REPLACE INTO tax_overrides (user_id, conto, immatricolazione, immobili) VALUES (?, ?, ?, ?)", target, cConto, cImm, cImmobili)
		respond(s, i, fmt.Sprintf("✅ **Tasse personalizzate salvate per <@%s>!**\nConto: € %d | Immatricolazione: € %d | Immobili: € %d", target, cConto, cImm, cImmobili))
		return
	}

	if strings.HasPrefix(customID, "modal_conferma_conto_") {
		banca := strings.TrimPrefix(customID, "modal_conferma_conto_")
		nomeRP := modalVal(data, 0)
		robloxUser := modalVal(data, 1)

		db.Exec("INSERT OR REPLACE INTO bank_accounts (user_id, owner_name, roblox_name, bank_name, balance, last_payday, last_tax_paid) VALUES (?, ?, ?, ?, 7000, ?, ?)",
			userID, nomeRP, robloxUser, banca, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

		respond(s, i, "🎉 **Conto Aperto!** Hai ricevuto **€ 7.000** di bonus.")

		embed := &discordgo.MessageEmbed{
			Title: "💳 NUOVA APERTURA CONTO BANCARIO",
			Color: 0x2ECC71,
			Description: fmt.Sprintf("**Richiedente:** <@%s>\n**Roblox:** `@%s`\n**Nome RP:** %s\n**Banca:** %s",
				userID, robloxUser, nomeRP, strings.ToUpper(banca)),
		}
		sendApproval(s, esitiChannel("apri-conto", i.ChannelID), embed, userID, "Accetta (Sì)", "Rifiuta (No)")
		return
	}

	switch customID {
	case "modal_porto_darmi":
		nomeRP := modalVal(data, 0)
		robloxUser := modalVal(data, 1)
		motivazione := modalVal(data, 2)

		db.Exec("INSERT OR REPLACE INTO porto_darmi (user_id, roblox_user, nome_rp, motivazione) VALUES (?, ?, ?, ?)", userID, robloxUser, nomeRP, motivazione)
		respond(s, i, "📜 **Richiesta Porto d'Armi Inviata!**")

		embed := &discordgo.MessageEmbed{
			Title: "🔫 RICHIESTA LICENZA PORTO D'ARMI",
			Color: 0xE67E22,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Richiedente", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
				{Name: "Roblox", Value: fmt.Sprintf("`@%s`", robloxUser), Inline: true},
				{Name: "Nome RP", Value: nomeRP, Inline: true},
				{Name: "Motivazione", Value: truncate(motivazione, 1000), Inline: false},
			},
		}
		sendApproval(s, esitiChannel("porto-darmi", i.ChannelID), embed, userID, "Accetta (Sì)", "Rifiuta (No)")

	case "modal_cittadinanza":
		nomeRP := modalVal(data, 0)
		cognomeRP := modalVal(data, 1)
		robloxUser := modalVal(data, 2)
		dataNascita := modalVal(data, 3)

		db.Exec("INSERT OR REPLACE INTO anagrafe (user_id, roblox_user, nome_rp, cognome_rp, data_nascita) VALUES (?, ?, ?, ?, ?)", userID, robloxUser, nomeRP, cognomeRP, dataNascita)
		respond(s, i, "🇮🇹 **Richiesta Cittadinanza Inviata!**")

		embed := &discordgo.MessageEmbed{
			Title: "🇮🇹 RICHIESTA CITTADINANZA / CARTA D'IDENTITÀ",
			Color: 0x3498DB,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Richiedente", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
				{Name: "Roblox", Value: fmt.Sprintf("`@%s`", robloxUser), Inline: true},
				{Name: "Nome RP", Value: nomeRP, Inline: true},
				{Name: "Cognome RP", Value: cognomeRP, Inline: true},
				{Name: "Nascita", Value: dataNascita, Inline: true},
			},
		}
		sendApproval(s, esitiChannel("cittadinanza", i.ChannelID), embed, userID, "Accetta (Sì)", "Rifiuta (No)")

	case "modal_immatricolazione":
		robloxUser := modalVal(data, 0)
		modelloAuto := modalVal(data, 1)
		targa := strings.ToUpper(strings.TrimSpace(modalVal(data, 2)))

		db.Exec("INSERT OR REPLACE INTO immatricolazioni (targa, user_id, roblox_user, modello_auto) VALUES (?, ?, ?, ?)", targa, userID, robloxUser, modelloAuto)
		respond(s, i, "🚗 **Immatricolazione Inviata!**")

		embed := &discordgo.MessageEmbed{
			Title: "🚗 RICHIESTA IMMATRICOLAZIONE VEICOLO",
			Color: 0x9B59B6,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Richiedente", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
				{Name: "Roblox", Value: fmt.Sprintf("`@%s`", robloxUser), Inline: true},
				{Name: "Veicolo", Value: modelloAuto, Inline: true},
				{Name: "Targa", Value: fmt.Sprintf("`%s`", targa), Inline: true},
			},
		}
		sendApproval(s, esitiChannel("immatricolazione", i.ChannelID), embed, userID, "Accetta (Sì)", "Rifiuta (No)")

	case "modal_registra_arresto":
		utenteArrestato := modalVal(data, 0)
		reati := modalVal(data, 1)
		prigione := modalVal(data, 2)
		cauzione := modalVal(data, 3)

		db.Exec("INSERT INTO arresti (agente_id, arrestato_roblox, reati, pena, cauzione, data) VALUES (?, ?, ?, ?, ?, ?)", userID, utenteArrestato, reati, prigione, cauzione, nowDate)

		embed := &discordgo.MessageEmbed{
			Title: "🚨 VERBALE D'ARRESTO UFFICIALE",
			Color: 0xE74C3C,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Agente", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
				{Name: "Arrestato", Value: fmt.Sprintf("`@%s`", utenteArrestato), Inline: true},
				{Name: "Pena", Value: prigione, Inline: true},
				{Name: "Cauzione", Value: cauzione, Inline: true},
				{Name: "Reati", Value: truncate(reati, 1000), Inline: false},
			},
		}
		respond(s, i, "🚨 **Verbale Registrato e archiviato nel Database Polizia!**")
		sendApproval(s, esitiChannel("registra-arresto", i.ChannelID), embed, userID, "Approva (Sì)", "Annulla (No)")

	case "modal_emetti_multa":
		utenteMultato := modalVal(data, 0)
		importo := modalVal(data, 1)
		motivo := modalVal(data, 2)
		targa := modalVal(data, 3)

		db.Exec("INSERT INTO multe (agente_id, multato_roblox, importo, motivo, targa, data) VALUES (?, ?, ?, ?, ?, ?)", userID, utenteMultato, importo, motivo, targa, nowDate)

		embed := &discordgo.MessageEmbed{
			Title: "📄 SANZIONE STRADALE / VERBALE MULTA",
			Color: 0xE74C3C,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Agente", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
				{Name: "Multato", Value: fmt.Sprintf("`@%s`", utenteMultato), Inline: true},
				{Name: "Importo", Value: fmt.Sprintf("€ %s", importo), Inline: true},
				{Name: "Targa", Value: targa, Inline: true},
				{Name: "Infrazione", Value: truncate(motivo, 1000), Inline: false},
			},
		}
		respond(s, i, "📄 **Multa Emessa e archiviata nel Database Polizia!**")
		sendApproval(s, esitiChannel("emetti-multa", i.ChannelID), embed, userID, "Approva (Sì)", "Annulla (No)")
	}
}
