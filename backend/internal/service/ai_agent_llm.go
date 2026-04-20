package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/cryptoagent"
	"wa-saas/backend/internal/model"
)

// ComposeAgentSystemPrompt junta tom base pt-BR + função + contexto do agente.
// voiceTTSActive: true quando o agente tem «Responder em áudio (TTS)» com provedor válido — evita o LLM negar envio de voz.
// O bloco de fluxos (quando existir) fica no início para o modelo priorizar produtos, preços e dados do negócio.
func ComposeAgentSystemPrompt(agentName, role, description string, voiceTTSActive bool, flowKnowledge string) string {
	var b strings.Builder
	if fk := strings.TrimSpace(flowKnowledge); fk != "" {
		b.WriteString("Base de conhecimento (fluxos publicados do negócio). ")
		b.WriteString("Trata este bloco como fonte principal: usa-o em todas as respostas automáticas e manuais sobre produtos, preços, serviços, horários, links e dados concretos do negócio; não contradigas o que aqui estiver definido nem inventes detalhes que não apareçam aqui.\n\n")
		b.WriteString(fk)
		b.WriteString("\n\n---\n\n")
	}
	b.WriteString("Responda sempre em português do Brasil (pt-BR), com tom profissional e cordial. ")
	b.WriteString("Use frases curtas e naturais, adequadas a conversas no WhatsApp. ")
	b.WriteString("Usa emojis com parcimónia: evita em várias frases seguidas; no máximo um emoji leve quando fizer sentido, ou nenhum — tom profissional primeiro. ")
	b.WriteString("Não uses Markdown nem asteriscos para negrito ou listas (*texto*, **texto**); escreve texto simples, como uma mensagem normal de telemóvel. ")
	b.WriteString("Quando te enviarem o histórico desta conversa no mesmo pedido, trata-o como memória: lembra-te do que já foi dito e continua o raciocínio (morada, orçamento, etc.). ")
	b.WriteString("Usa o nome do cliente só quando soar natural: saudação ou recomeço de conversa, proposta/orçamento formal, ou quando ele perguntar se lembras quem ele é — não repitas o nome em todas as mensagens. ")
	b.WriteString("Não inventes diminutivos nem apelidos carinhosos do nome (ex.: acrescentar «zinho») a menos que o próprio cliente use esse tratamento. ")
	b.WriteString("Se já cumprimentaste nas tuas mensagens anteriores nesta conversa (ou no histórico), não reabres com «Olá» nem com o nome no início — responde direto ao assunto.\n\n")
	if voiceTTSActive {
		b.WriteString("Respostas em áudio (TTS): com esta opção ligada, o sistema converte automaticamente o teu texto em mensagem de voz no WhatsApp. ")
		b.WriteString("Não digas que não consegues enviar áudios, que só respondes por texto, ou que não tens microfone. ")
		b.WriteString("Não recuses pedidos do tipo «manda áudio» ou «fala em voz alta»: responde ao conteúdo normalmente; a plataforma trata da entrega em voz. ")
		b.WriteString("Mensagens muito curtas e genéricas podem ir só em texto; respostas longas ou com orçamento, agendamento e dados concretos tendem a ir em nota de voz. ")
		b.WriteString("Em respostas mais extensas podes opcionalmente intercalar as etiquetas [PAUSA], [HESITA] e [GAGUEJA] entre frases — com voz Gemini TTS são usadas para ritmo natural de nota de voz (não digas o nome das etiquetas em voz alta).\n\n")
	}
	if n := strings.TrimSpace(agentName); n != "" {
		b.WriteString("Nome do assistente: ")
		b.WriteString(n)
		b.WriteString("\n\n")
	}
	if r := strings.TrimSpace(role); r != "" {
		b.WriteString("Função / papel: ")
		b.WriteString(r)
		b.WriteString("\n\n")
	}
	if d := strings.TrimSpace(description); d != "" {
		b.WriteString("Contexto e instruções:\n")
		b.WriteString(d)
	}
	return strings.TrimSpace(b.String())
}

// BuildLLMFromAgent desencripta a chave e instancia o cliente LLM.
// Se db != nil, agrega texto dos fluxos publicados ligados a este agente ao system prompt.
func BuildLLMFromAgent(db *gorm.DB, encryptionKey string, a *model.AIAgent) (LLM, error) {
	if a == nil {
		return nil, fmt.Errorf("agente nil")
	}
	rawKey, err := cryptoagent.Decrypt(a.APIKeyCipher, encryptionKey)
	if err != nil {
		return nil, err
	}
	voiceTTS := a.VoiceReplyEnabled && NormalizeTTSProvider(a.TTSProvider) != TTSProviderNone
	var flowBlock string
	if db != nil {
		flowBlock, err = AggregatedFlowKnowledgeForAgent(db, a.WorkspaceID, a.ID)
		if err != nil {
			return nil, err
		}
	}
	sys := ComposeAgentSystemPrompt(a.Name, a.Role, a.Description, voiceTTS, flowBlock)
	switch strings.ToLower(strings.TrimSpace(a.Provider)) {
	case "gemini":
		return NewGeminiClient(rawKey, strings.TrimSpace(a.Model), sys), nil
	case "openai":
		return NewOpenAIClient(rawKey, strings.TrimSpace(a.Model), sys), nil
	default:
		return nil, fmt.Errorf("provider desconhecido: %q", a.Provider)
	}
}

// WorkspaceAutoReplyLLM devolve o LLM do agente marcado para WhatsApp neste workspace, ou nil se não houver.
func WorkspaceAutoReplyLLM(db *gorm.DB, encryptionKey string, workspaceID uuid.UUID) (LLM, error) {
	a, err := WorkspaceAutoReplyAgent(db, workspaceID)
	if err != nil || a == nil {
		return nil, err
	}
	if strings.TrimSpace(a.APIKeyCipher) == "" {
		return nil, nil
	}
	return BuildLLMFromAgent(db, encryptionKey, a)
}

// modelOrDefault devolve o modelo do agente se preenchido; caso contrário o default de config.
func modelOrDefault(agentModel, cfgDefault string) string {
	if s := strings.TrimSpace(agentModel); s != "" {
		return s
	}
	return strings.TrimSpace(cfgDefault)
}

// AutoReplyLLMWithAgentAndFlowKnowledgeFromEnv monta o mesmo system prompt que BuildLLMFromAgent (perfil + fluxos publicados),
// mas usando GEMINI_API_KEY / OPENAI_API_KEY do ambiente em vez da chave encriptada do agente.
// Serve quando o webhook ainda está no LLM global: agente sem api_key na BD, falha de descriptografia, ou APP_ENCRYPTION_KEY vazio
// — desde que exista chave LLM no .env compatível com o provider do agente.
func AutoReplyLLMWithAgentAndFlowKnowledgeFromEnv(db *gorm.DB, cfg *config.Config, workspaceID uuid.UUID) LLM {
	if db == nil || cfg == nil || workspaceID == uuid.Nil {
		return nil
	}
	a, err := WorkspaceAutoReplyAgent(db, workspaceID)
	if err != nil || a == nil {
		return nil
	}
	flowBlock, err := AggregatedFlowKnowledgeForAgent(db, workspaceID, a.ID)
	if err != nil {
		flowBlock = ""
	}
	voiceTTS := a.VoiceReplyEnabled && NormalizeTTSProvider(a.TTSProvider) != TTSProviderNone
	sys := ComposeAgentSystemPrompt(a.Name, a.Role, a.Description, voiceTTS, flowBlock)

	provider := strings.ToLower(strings.TrimSpace(a.Provider))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(cfg.LLMProvider))
	}
	switch provider {
	case "gemini":
		key := strings.TrimSpace(cfg.GeminiAPIKey)
		if key == "" {
			return nil
		}
		return NewGeminiClient(key, modelOrDefault(a.Model, cfg.GeminiModel), sys)
	case "openai":
		key := strings.TrimSpace(cfg.OpenAIAPIKey)
		if key == "" {
			return nil
		}
		return NewOpenAIClient(key, modelOrDefault(a.Model, cfg.OpenAIModel), sys)
	default:
		return nil
	}
}

// WorkspaceAutoReplyAgent carrega o agente ativo para auto-resposta WhatsApp (um por workspace).
func WorkspaceAutoReplyAgent(db *gorm.DB, workspaceID uuid.UUID) (*model.AIAgent, error) {
	var a model.AIAgent
	err := db.Where("workspace_id = ? AND active = ? AND use_for_whatsapp_auto_reply = ?",
		workspaceID, true, true).First(&a).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// WorkspaceAutoReplyNoLLMReason explica porque WorkspaceAutoReplyLLM devolveria (nil, nil) — útil para logs.
// O carregamento real exige active=true, use_for_whatsapp_auto_reply=true e api_key_cipher preenchido.
func WorkspaceAutoReplyNoLLMReason(db *gorm.DB, workspaceID uuid.UUID) string {
	if db == nil || workspaceID == uuid.Nil {
		return "workspace_id inválido"
	}
	var agents []model.AIAgent
	if err := db.Where("workspace_id = ? AND use_for_whatsapp_auto_reply = ?", workspaceID, true).
		Order("updated_at DESC").Find(&agents).Error; err != nil {
		return "db: " + err.Error()
	}
	if len(agents) == 0 {
		return "nenhum agente com use_for_whatsapp_auto_reply=true (marca no painel e grava)"
	}
	for _, a := range agents {
		if !a.Active {
			return "agente «" + a.Name + "» tem WhatsApp auto-resposta mas active=false — activa o agente"
		}
		if strings.TrimSpace(a.APIKeyCipher) == "" {
			return "agente «" + a.Name + "» sem chave LLM gravada — cola a API key no agente e guarda (PATCH com api_key)"
		}
	}
	return "agente(s) com WhatsApp marcado mas critérios de LLM não reunidos"
}

// ClearOtherWhatsAppAutoReplyAgents desmarca outros agentes do mesmo workspace (máx. um ativo).
func ClearOtherWhatsAppAutoReplyAgents(db *gorm.DB, workspaceID uuid.UUID, exceptAgentID uuid.UUID) error {
	q := db.Model(&model.AIAgent{}).Where("workspace_id = ? AND use_for_whatsapp_auto_reply = ?", workspaceID, true)
	if exceptAgentID != uuid.Nil {
		q = q.Where("id != ?", exceptAgentID)
	}
	return q.Update("use_for_whatsapp_auto_reply", false).Error
}
