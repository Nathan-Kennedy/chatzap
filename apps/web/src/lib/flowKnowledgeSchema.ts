import { z } from 'zod'

/** Alinhado com backend `model.FlowKnowledge` / validação em `flow_knowledge.go`. */
export const flowTimeSlotSchema = z.object({
  dias_semana: z.array(z.number().int().min(0).max(6)),
  inicio: z.string().max(16),
  fim: z.string().max(16),
})

export const flowKnowledgeSchema = z.object({
  produtos: z.array(
    z.object({
      nome: z.string().max(256),
      descricao: z.string().max(4000),
      preco_referencia: z.string().max(64),
    })
  ),
  servicos: z.array(
    z.object({
      nome: z.string().max(256),
      descricao: z.string().max(4000),
      duracao_estimada: z.string().max(128),
    })
  ),
  disponibilidade: z.object({
    slots_texto: z.string().max(8000),
    observacoes_horario: z.string().max(4000),
    slots: z.array(flowTimeSlotSchema),
  }),
  links: z.array(
    z.object({
      rotulo: z.string().max(256),
      url: z.string().max(2048),
    })
  ),
  imagens: z.array(
    z.object({
      url: z.string().max(2048),
      legenda: z.string().max(512),
    })
  ),
  notas_gerais: z.string().max(16000),
})

export const flowEditFormSchema = z.object({
  name: z.string().min(1, 'Nome é obrigatório').max(256),
  description: z.string().max(20000),
  published: z.boolean(),
  agent_id: z.string(),
  knowledge: flowKnowledgeSchema,
})

export type FlowEditFormValues = z.infer<typeof flowEditFormSchema>
export type FlowKnowledgeFormValues = z.infer<typeof flowKnowledgeSchema>

export function emptyFlowKnowledge(): FlowKnowledgeFormValues {
  return {
    produtos: [],
    servicos: [],
    disponibilidade: {
      slots_texto: '',
      observacoes_horario: '',
      slots: [],
    },
    links: [],
    imagens: [],
    notas_gerais: '',
  }
}

/** Remove linhas vazias antes de enviar ao servidor (o back também ignora). */
export function sanitizeFlowKnowledge(k: FlowKnowledgeFormValues): FlowKnowledgeFormValues {
  const produtos = k.produtos.filter(
    (p) =>
      p.nome.trim() !== '' ||
      p.descricao.trim() !== '' ||
      p.preco_referencia.trim() !== ''
  )
  const servicos = k.servicos.filter(
    (s) =>
      s.nome.trim() !== '' ||
      s.descricao.trim() !== '' ||
      s.duracao_estimada.trim() !== ''
  )
  const links = k.links.filter((l) => l.rotulo.trim() !== '' || l.url.trim() !== '')
  const imagens = k.imagens.filter((i) => i.url.trim() !== '' || i.legenda.trim() !== '')
  const slots = k.disponibilidade.slots.filter(
    (sl) =>
      sl.dias_semana.length > 0 ||
      sl.inicio.trim() !== '' ||
      sl.fim.trim() !== ''
  )
  return {
    ...k,
    produtos,
    servicos,
    links,
    imagens,
    disponibilidade: {
      ...k.disponibilidade,
      slots,
    },
  }
}
