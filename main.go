# 🏛️ Rome Italy 2.0 - Custom Discord RP Bot

Un bot Discord avanzato, modulare ed interamente basato su **Interfacce Grafiche (Buttons & Select Menus)** e **Moduli Pop-up (Modals)**, progettato appositamente per la community Roleplay su **Emergency Hamburg (Roblox)** di *Rome Italy 2.0*.

---

## 🌟 Caratteristiche Principali

* 🎛️ **Configurazione Hub Unificata (`/setup server`):** Pannello amministrativo centrale per gestire ruoli fazione, canali di log e stipendi dinamici senza toccare il codice.
* 💳 **Sistema Bancario Reale & FinTech:** Gestione aperture conti su banche italiane tradizionali (*Banca Sella, Intesa Sanpaolo, UniCredit, Poste Italiane, ecc.*) ed online (*Revolut, HYPE, N26, BBVA, ecc.*).
* 💼 **Busta Paga Automatica & Dinamica:** Riconoscimento automatico dei ruoli fazione con erogazione dello stipendio personalizzato e blocco temporale (*cooldown*) settimanale.
* 🚨 **Banca Dati Polizia & Registro Arresti:** Form per verbali d'arresto immediati, calcolo cauzioni, verbali per contravvenzioni e storico sanzioni penali.
* 🇮🇹 **Anagrafe & Motorizzazione (PRA):** Registrazione automatizzata di carte d'identità, licenze per armi da fuoco e immatricolazione veicoli con targhe personalizzate.
* 🛡️ **Moderazione Staff & Sanzioni RP:** Pannello dedicato alla gestione di Warn per FailRP/RDM/VDM e Ban temporanei dalle sessioni RP con registro clip.

---

## 🎛️ Configurazione Server Administrator: `/setup server`

I Fondatori e gli Admin possono eseguire `/setup server` per aprire il pannello di controllo centrale. Tramite menu a tendina e interfacce interattive è possibile configurare:

### 🟢 A. Economia & Banche
* **Imposta Stipendi per Ruolo:** Assegnazione di un importo esatto in Euro (€) associato a uno specifico Ruolo Discord (es. `@Carabiniere`, `@Polizia`, `@EMS`).
* **Canale Log Banche:** Canale di tracciamento per bonifici, transazioni e riscossione buste paga.
* **Cooldown Busta Paga:** Blocco temporale configurabile per il riscatto dello stipendio (Default: *7 giorni*).

### 🟦 B. Anagrafe & Motorizzazione
* **Ruolo Cittadino:** Ruolo assegnato automaticamente al completamento del form anagrafico.
* **Canale Registro Cittadinanze:** Registro pubblico/amministrativo delle carte d'identità.
* **Canale PRA:** Pubblico Registro Automobilistico per l'archiviazione dei veicoli e delle targhe registrate.

### 🔴 C. Forze dell'Ordine & Giustizia
* **Ruoli Forze dell'Ordine:** Abilitazione dei ruoli Discord autorizzati ad accedere ai form d'arresto e stesura multe.
* **Canale Registro Arresti:** Pubblicazione automatica dei verbali d'arresto formattati.
* **Canale Log Multe:** Tracciamento delle sanzioni pecuniarie emesse.

### 🟧 D. Staff & Moderazione
* **Ruoli Staff:** Ruoli abilitati all'utilizzo dei comandi sanzionatori.
* **Canale Log Sanzioni:** Registro riservato per i report e le clip di prova allegate.

---

## ⚙️ Comandi di Setup Pannelli Singoli

Il bot mette a disposizione una suite di comandi `/setup [modulo]` per inviare messaggi Embed permanenti dotati di bottoni interattivi all'interno dei canali dedicati.

---

### 💳 1. Settore Economia & Banche

#### `/setup apri-conto`
* **Pannello:** `[ 💳 Apri Conto Corrente ]`
* **Flusso:**
  1. Apre un **Select Menu** per la scelta dell'istituto bancario:
     * *Banche Tradizionali:* Banca Sella, Poste Italiane, Intesa Sanpaolo, UniCredit, BPER Banca, Banco BPM, Monte dei Paschi di Siena, Crédit Agricole.
     * *Banche Online / FinTech:* Revolut, HYPE, N26, FinecoBank, BBVA Italia, Buddybank.
  2. Apre un **Modal (Popup)** per confermare *Nome & Cognome RP* e creare il conto.

#### `/setup bonifico`
* **Pannello:** `[ 💸 Invia Bonifico ]`
* **Flusso:** Apre un **Modal** per inserire *Destinatario*, *Banca*, *Importo (€)* e *Causale*.

#### `/setup busta-paga`
* **Pannello:** `[ 💼 Riscuoti Busta Paga ]`
* **Flusso:** 
  1. Rileva automaticamente i ruoli Discord dell'utente.
  2. Accredita lo stipendio configurato tramite `/setup server` sul conto dell'utente (assegnando la paga del ruolo più alto in caso di ruoli multipli).
  3. Applica il blocco temporale settimanale impedendo riscossioni multiple prima della scadenza.

---

### 🇮🇹 2. Settore Anagrafe, Documenti & Veicoli

#### `/setup cittadinanza`
* **Pannello:** `[ 🇮🇹 Richiedi Cittadinanza ]`
* **Flusso:** Apre un **Modal** per compilare *Nome RP*, *Cognome RP* e *Data di Nascita*.

#### `/setup porto-darmi`
* **Pannello:** `[ 📜 Richiedi Porto d'Armi ]`
* **Flusso:** Apre un **Modal** per selezionare la tipologia di licenza (*Uso Sportivo / Difesa Personale*) e inserire la motivazione.

#### `/setup immatricolazione`
* **Pannello:** `[ 🚘 Immatricola Veicolo ]`
* **Flusso:** Apre un **Modal** per registrare *Modello Veicolo*, *Targa Desiderata* e *Colore*.

---

### 🚨 3. Settore Polizia & Giustizia

#### `/setup registra-arresto`
* **Pannello:** `[ 🚨 Registra Arresto ]` *(Riservato alle Forze dell'Ordine)*
* **Flusso:** Apre un **Modal** per verbalizzare:
  * *Utente / ID Roblox dell'arrestato*
  * *Reati commessi*
  * *Anni / Mesi di prigione*
  * *Cauzione (€)*
  * *Link Prova Video / Clip*
* Invia automaticamente il verbale nel canale **Registro Arresti**.

#### `/setup emetti-multa`
* **Pannello:** `[ 📝 Emetti Multa ]`
* **Flusso:** Apre un **Modal** per compilare *Cittadino Sanzionato*, *Importo (€)* e *Articolo del Codice della Strada/Motivo*.

#### `/setup registro-arresti`
* **Pannello:** `[ 📋 Consulta Registro Arresti ]` | `[ 🔍 Cerca Fedina Penale ]`
* **Flusso:** Permette la consultazione pubblica ed immediata dei precedenti penali e dei fermi subiti dai cittadini.

---

### ⚠️ 4. Settore Moderazione Staff

#### `/setup staff-panel`
* **Pannello:** `[ ⚠️ Emetti Warn ]` | `[ ⛔ Ban RP ]` | `[ 🟢 Unban RP ]`
* **Flusso:** Moduli riservati ai moderatori per applicare o revocare sanzioni (FailRP, RDM, VDM) inserendo durata, motivazione e link alle clip di prova.

---

## 🛠️ Tech Stack & Requisiti

* **Language:** Go / Discord.js (v14) / Python
* **UI Components:** Discord Buttons, Select Menus, Modals (Interactions v2)
* **Database:** SQLite / PostgreSQL / MongoDB (per il tracciamento di saldi, cooldown e fedine penali)

---

## 📄 Licenza

Questo progetto è distribuito sotto licenza MIT. Realizzato per la community **Rome Italy 2.0 (Emergency Hamburg RP)**.
