package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/cryptoagent"
	"wa-saas/backend/internal/model"
)

// ComposeAgentSystemPrompt junta tom base pt-BR + função + contexto do agente.
func ComposeAgentSystemPrompt(agentName, role, description string) string {
	var b strings.Builder
	b.WriteString("Responda sempre em português do Brasil (pt-BR), com tom profissional e cordial. ")
	b.WriteString("Use frases curtas e naturais, adequadas a conversas no WhatsApp. ")
	b.WriteString("Usa emojis com parcimónia: evita em várias frases seguidas; no máximo um emoji leve quando fizer sentido, ou nenhum — tom profissional primeiro. ")
	b.WriteString("Não uses Markdown nem asteriscos para negrito ou listas (*texto*, **texto**); escreve texto simples, como uma mensagem normal de telemóvel. ")
	b.WriteString("Quando te enviarem o histórico desta conversa no mesmo pedido, trata-o como memória: lembra-te do que já foi dito e continua o raciocínio (morada, orçamento, etc.). ")
	b.WriteString("Usa o nome do cliente só quando soar natural: saudação ou recomeço de conversa, proposta/orçamento formal, ou quando ele perguntar se lembras quem ele é — não repitas o nome em todas as mensagens. ")
	b.WriteString("Não inventes diminutivos nem apelidos carinhosos do nome (ex.: acrescentar «zinho») a menos que o próprio cliente use esse tratamento. ")
	b.WriteString("Se já cumprimentaste nas tuas mensagens anteriores nesta conversa (ou no histórico), não reabres com «Olá» nem com o nome no início — responde direto ao assunto.\n\n")
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
func BuildLLMFromAgent(encryptionKey string, a *model.AIAgent) (LLM, error) {
	if a == nil {
		return nil, fmt.Errorf("agente nil")
	}
	rawKey, err := cryptoagent.Decrypt(a.APIKeyCipher, encryptionKey)
	if err != nil {
		return nil, err
	}
	sys := ComposeAgentSystemPrompt(a.Name, a.Role, a.Description)
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
	return BuildLLMFromAgent(encryptionKey, a)
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

// ClearOtherWhatsAppAutoReplyAgents desmarca outros agentes do mesmo workspace (máx. um ativo).
func ClearOtherWhatsAppAutoReplyAgents(db *gorm.DB, workspaceID uuid.UUID, exceptAgentID uuid.UUID) error {
	q := db.Model(&model.AIAgent{}).Where("workspace_id = ? AND use_for_whatsapp_auto_reply = ?", workspaceID, true)
	if exceptAgentID != uuid.Nil {
		q = q.Where("id != ?", exceptAgentID)
	}
	return q.Update("use_for_whatsapp_auto_reply", false).Error
}
