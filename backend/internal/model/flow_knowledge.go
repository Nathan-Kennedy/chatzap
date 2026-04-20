package model

// FlowKnowledge documento JSON guardado em flows.knowledge_json (schema estável front/back).
type FlowKnowledge struct {
	Produtos         []FlowProduct        `json:"produtos"`
	Servicos         []FlowService        `json:"servicos"`
	Disponibilidade  FlowAvailability     `json:"disponibilidade"`
	Links            []FlowLink           `json:"links"`
	Imagens          []FlowImage          `json:"imagens"`
	NotasGerais      string               `json:"notas_gerais"`
}

type FlowProduct struct {
	Nome            string `json:"nome"`
	Descricao       string `json:"descricao"`
	PrecoReferencia string `json:"preco_referencia"`
}

type FlowService struct {
	Nome             string `json:"nome"`
	Descricao        string `json:"descricao"`
	DuracaoEstimada  string `json:"duracao_estimada"`
}

type FlowAvailability struct {
	SlotsTexto          string          `json:"slots_texto"`
	ObservacoesHorario  string          `json:"observacoes_horario"`
	Slots               []FlowTimeSlot  `json:"slots"`
}

type FlowTimeSlot struct {
	DiasSemana []int  `json:"dias_semana"`
	Inicio     string `json:"inicio"`
	Fim        string `json:"fim"`
}

type FlowLink struct {
	Rotulo string `json:"rotulo"`
	URL    string `json:"url"`
}

type FlowImage struct {
	URL    string `json:"url"`
	Legenda string `json:"legenda"`
}
