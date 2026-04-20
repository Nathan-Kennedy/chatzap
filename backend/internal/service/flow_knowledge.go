package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
)

const (
	flowKnowledgeJSONMaxBytes       = 512 * 1024
	AggregatedFlowKnowledgeMaxRunes = 12000
	maxFlowProducts                 = 80
	maxFlowServices                 = 80
	maxFlowLinks                    = 40
	maxFlowImages                   = 40
	maxFlowTimeSlots                = 50
	maxNameLen                      = 256
	maxDescLen                      = 4000
	maxPrecoLen                     = 64
	maxDuracaoLen                   = 128
	maxSlotsTextoLen                = 8000
	maxObsHorarioLen                = 4000
	maxNotasGeraisLen               = 16000
	maxURLLen                       = 2048
	maxRotuloLen                    = 256
	maxLegendaLen                   = 512
)

var diasSemanaNome = []string{"dom", "seg", "ter", "qua", "qui", "sex", "sáb"}

// ParseFlowKnowledgeJSON deserializa e valida; string vazia ou "{}" → zero value válido.
func ParseFlowKnowledgeJSON(raw string) (model.FlowKnowledge, error) {
	s := strings.TrimSpace(raw)
	if s == "" || s == "{}" {
		return model.FlowKnowledge{}, nil
	}
	if len(s) > flowKnowledgeJSONMaxBytes {
		return model.FlowKnowledge{}, fmt.Errorf("knowledge_json demasiado grande (máx. %d bytes)", flowKnowledgeJSONMaxBytes)
	}
	var k model.FlowKnowledge
	if err := json.Unmarshal([]byte(s), &k); err != nil {
		return model.FlowKnowledge{}, fmt.Errorf("knowledge_json inválido: %w", err)
	}
	if err := ValidateFlowKnowledge(&k); err != nil {
		return model.FlowKnowledge{}, err
	}
	return k, nil
}

// ValidateFlowKnowledge limita cardinalidade e tamanhos por campo.
func ValidateFlowKnowledge(k *model.FlowKnowledge) error {
	if k == nil {
		return nil
	}
	if len(k.Produtos) > maxFlowProducts {
		return fmt.Errorf("máximo %d produtos", maxFlowProducts)
	}
	for i, p := range k.Produtos {
		if strings.TrimSpace(p.Nome) == "" && strings.TrimSpace(p.Descricao) == "" && strings.TrimSpace(p.PrecoReferencia) == "" {
			continue // linha vazia ignorada
		}
		if err := checkLen("produtos["+strconv.Itoa(i)+"].nome", p.Nome, 1, maxNameLen); err != nil {
			return err
		}
		if err := checkLenOpt("produtos["+strconv.Itoa(i)+"].descricao", p.Descricao, maxDescLen); err != nil {
			return err
		}
		if err := checkLenOpt("produtos["+strconv.Itoa(i)+"].preco_referencia", p.PrecoReferencia, maxPrecoLen); err != nil {
			return err
		}
	}
	if len(k.Servicos) > maxFlowServices {
		return fmt.Errorf("máximo %d serviços", maxFlowServices)
	}
	for i, s := range k.Servicos {
		if strings.TrimSpace(s.Nome) == "" && strings.TrimSpace(s.Descricao) == "" && strings.TrimSpace(s.DuracaoEstimada) == "" {
			continue
		}
		if err := checkLen("servicos["+strconv.Itoa(i)+"].nome", s.Nome, 1, maxNameLen); err != nil {
			return err
		}
		if err := checkLenOpt("servicos["+strconv.Itoa(i)+"].descricao", s.Descricao, maxDescLen); err != nil {
			return err
		}
		if err := checkLenOpt("servicos["+strconv.Itoa(i)+"].duracao_estimada", s.DuracaoEstimada, maxDuracaoLen); err != nil {
			return err
		}
	}
	if len(k.Links) > maxFlowLinks {
		return fmt.Errorf("máximo %d links", maxFlowLinks)
	}
	for i, l := range k.Links {
		if strings.TrimSpace(l.URL) == "" && strings.TrimSpace(l.Rotulo) == "" {
			continue
		}
		if err := checkLen("links["+strconv.Itoa(i)+"].rotulo", l.Rotulo, 1, maxRotuloLen); err != nil {
			return err
		}
		if err := checkLen("links["+strconv.Itoa(i)+"].url", l.URL, 1, maxURLLen); err != nil {
			return err
		}
	}
	if len(k.Imagens) > maxFlowImages {
		return fmt.Errorf("máximo %d imagens", maxFlowImages)
	}
	for i, im := range k.Imagens {
		if strings.TrimSpace(im.URL) == "" && strings.TrimSpace(im.Legenda) == "" {
			continue
		}
		if err := checkLen("imagens["+strconv.Itoa(i)+"].url", im.URL, 1, maxURLLen); err != nil {
			return err
		}
		if err := checkLenOpt("imagens["+strconv.Itoa(i)+"].legenda", im.Legenda, maxLegendaLen); err != nil {
			return err
		}
	}
	if err := checkLenOpt("disponibilidade.slots_texto", k.Disponibilidade.SlotsTexto, maxSlotsTextoLen); err != nil {
		return err
	}
	if err := checkLenOpt("disponibilidade.observacoes_horario", k.Disponibilidade.ObservacoesHorario, maxObsHorarioLen); err != nil {
		return err
	}
	if len(k.Disponibilidade.Slots) > maxFlowTimeSlots {
		return fmt.Errorf("máximo %d blocos de horário", maxFlowTimeSlots)
	}
	for i, sl := range k.Disponibilidade.Slots {
		for _, d := range sl.DiasSemana {
			if d < 0 || d > 6 {
				return fmt.Errorf("disponibilidade.slots[%d]: dia %d inválido (use 0–6, 0=domingo)", i, d)
			}
		}
		if err := checkLenOpt("disponibilidade.slots["+strconv.Itoa(i)+"].inicio", sl.Inicio, 16); err != nil {
			return err
		}
		if err := checkLenOpt("disponibilidade.slots["+strconv.Itoa(i)+"].fim", sl.Fim, 16); err != nil {
			return err
		}
	}
	if err := checkLenOpt("notas_gerais", k.NotasGerais, maxNotasGeraisLen); err != nil {
		return err
	}
	return nil
}

func checkLen(field, s string, minRune, maxRune int) error {
	n := utf8.RuneCountInString(strings.TrimSpace(s))
	if minRune > 0 && n < minRune {
		return fmt.Errorf("%s é obrigatório", field)
	}
	if n > maxRune {
		return fmt.Errorf("%s: máximo %d caracteres", field, maxRune)
	}
	return nil
}

func checkLenOpt(field, s string, maxRune int) error {
	if utf8.RuneCountInString(s) > maxRune {
		return fmt.Errorf("%s: máximo %d caracteres", field, maxRune)
	}
	return nil
}

// FormatFlowKnowledgeForPrompt texto legível para system prompt (um fluxo). Vazio se não houver conteúdo.
func FormatFlowKnowledgeForPrompt(flowName string, k model.FlowKnowledge) string {
	fn := strings.TrimSpace(flowName)
	if fn == "" {
		fn = "Fluxo"
	}
	var body strings.Builder
	if len(k.Produtos) > 0 {
		body.WriteString("Produtos:\n")
		for _, p := range k.Produtos {
			line := strings.TrimSpace(p.Nome)
			if line == "" {
				continue
			}
			body.WriteString("- ")
			body.WriteString(line)
			if d := strings.TrimSpace(p.Descricao); d != "" {
				body.WriteString(" — ")
				body.WriteString(d)
			}
			if pr := strings.TrimSpace(p.PrecoReferencia); pr != "" {
				body.WriteString(" (ref. ")
				body.WriteString(pr)
				body.WriteString(")")
			}
			body.WriteByte('\n')
		}
	}
	if len(k.Servicos) > 0 {
		body.WriteString("Serviços:\n")
		for _, sv := range k.Servicos {
			line := strings.TrimSpace(sv.Nome)
			if line == "" {
				continue
			}
			body.WriteString("- ")
			body.WriteString(line)
			if d := strings.TrimSpace(sv.Descricao); d != "" {
				body.WriteString(" — ")
				body.WriteString(d)
			}
			if du := strings.TrimSpace(sv.DuracaoEstimada); du != "" {
				body.WriteString(" (duração ref. ")
				body.WriteString(du)
				body.WriteString(")")
			}
			body.WriteByte('\n')
		}
	}
	disp := k.Disponibilidade
	if t := strings.TrimSpace(disp.SlotsTexto); t != "" {
		body.WriteString("Disponibilidade / horários:\n")
		body.WriteString(t)
		body.WriteByte('\n')
	}
	if len(disp.Slots) > 0 {
		body.WriteString("Horários (blocos):\n")
		for _, sl := range disp.Slots {
			var days []string
			for _, d := range sl.DiasSemana {
				if d >= 0 && d < len(diasSemanaNome) {
					days = append(days, diasSemanaNome[d])
				}
			}
			if len(days) == 0 && sl.Inicio == "" && sl.Fim == "" {
				continue
			}
			body.WriteString("- ")
			if len(days) > 0 {
				body.WriteString(strings.Join(days, ", "))
				body.WriteString(": ")
			}
			body.WriteString(strings.TrimSpace(sl.Inicio))
			body.WriteString(" – ")
			body.WriteString(strings.TrimSpace(sl.Fim))
			body.WriteByte('\n')
		}
	}
	if o := strings.TrimSpace(disp.ObservacoesHorario); o != "" {
		body.WriteString("Observações de horário: ")
		body.WriteString(o)
		body.WriteByte('\n')
	}
	if len(k.Links) > 0 {
		body.WriteString("Links úteis:\n")
		for _, l := range k.Links {
			if strings.TrimSpace(l.URL) == "" {
				continue
			}
			body.WriteString("- ")
			body.WriteString(strings.TrimSpace(l.Rotulo))
			body.WriteString(": ")
			body.WriteString(strings.TrimSpace(l.URL))
			body.WriteByte('\n')
		}
	}
	if len(k.Imagens) > 0 {
		body.WriteString("Imagens (URLs):\n")
		for _, im := range k.Imagens {
			if strings.TrimSpace(im.URL) == "" {
				continue
			}
			body.WriteString("- ")
			body.WriteString(strings.TrimSpace(im.URL))
			if leg := strings.TrimSpace(im.Legenda); leg != "" {
				body.WriteString(" — ")
				body.WriteString(leg)
			}
			body.WriteByte('\n')
		}
	}
	if n := strings.TrimSpace(k.NotasGerais); n != "" {
		body.WriteString("Notas / FAQ / políticas:\n")
		body.WriteString(n)
		body.WriteByte('\n')
	}
	trimmed := strings.TrimSpace(body.String())
	if trimmed == "" {
		return ""
	}
	return "## " + fn + "\n" + trimmed
}

// AggregatedFlowKnowledgeForAgent concatena fluxos publicados do agente (ordem updated_at ASC).
func AggregatedFlowKnowledgeForAgent(db *gorm.DB, workspaceID, agentID uuid.UUID) (string, error) {
	if db == nil || workspaceID == uuid.Nil || agentID == uuid.Nil {
		return "", nil
	}
	var flows []model.Flow
	if err := db.Where("workspace_id = ? AND agent_id = ? AND published = ?", workspaceID, agentID, true).
		Order("updated_at ASC").Find(&flows).Error; err != nil {
		return "", err
	}
	if len(flows) == 0 {
		return "", nil
	}
	var parts []string
	for _, f := range flows {
		k, err := ParseFlowKnowledgeJSON(f.KnowledgeJSON)
		if err != nil {
			return "", err
		}
		block := FormatFlowKnowledgeForPrompt(f.Name, k)
		if block != "" {
			parts = append(parts, block)
		}
	}
	if len(parts) == 0 {
		return "", nil
	}
	joined := strings.Join(parts, "\n\n")
	return truncateRunes(joined, AggregatedFlowKnowledgeMaxRunes), nil
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) > max {
		return string(r[:max]) + "… [truncado]"
	}
	return s
}

// FlowKnowledgePromptPreview bloco formatado para um único fluxo (GET detalhe).
func FlowKnowledgePromptPreview(flowName string, rawJSON string) (string, error) {
	k, err := ParseFlowKnowledgeJSON(rawJSON)
	if err != nil {
		return "", err
	}
	return FormatFlowKnowledgeForPrompt(flowName, k), nil
}
