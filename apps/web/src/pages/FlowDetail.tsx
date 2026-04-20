import { useEffect } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Controller, useFieldArray, useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import {
  ArrowLeft,
  Loader2,
  Plus,
  Trash2,
} from 'lucide-react'
import { api, unwrapEnvelope } from '@/lib/api'
import {
  emptyFlowKnowledge,
  flowEditFormSchema,
  sanitizeFlowKnowledge,
  type FlowEditFormValues,
} from '@/lib/flowKnowledgeSchema'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { toast } from 'sonner'
import { ApiEnvelopeError } from '@/types/api'

type FlowDetailDTO = {
  id: string
  name: string
  description: string
  published: boolean
  agent_id: string | null
  knowledge?: FlowEditFormValues['knowledge']
  prompt_preview: string
}

type AgentListRow = { id: string; name: string }

async function fetchFlow(id: string): Promise<FlowDetailDTO> {
  const res = await api.get<unknown>(`/flows/${id}`)
  const { data } = unwrapEnvelope<FlowDetailDTO>(res)
  return data
}

async function fetchAgents(): Promise<AgentListRow[]> {
  const res = await api.get<unknown>('/agents')
  const { data } = unwrapEnvelope<AgentListRow[]>(res)
  return data
}

function dtoToForm(d: FlowDetailDTO): FlowEditFormValues {
  const e = emptyFlowKnowledge()
  const k = d.knowledge ?? e
  return {
    name: d.name,
    description: d.description ?? '',
    published: d.published,
    agent_id: d.agent_id ?? '',
    knowledge: {
      produtos: k.produtos ?? e.produtos,
      servicos: k.servicos ?? e.servicos,
      disponibilidade: {
        slots_texto: k.disponibilidade?.slots_texto ?? '',
        observacoes_horario: k.disponibilidade?.observacoes_horario ?? '',
        slots: k.disponibilidade?.slots ?? e.disponibilidade.slots,
      },
      links: k.links ?? e.links,
      imagens: k.imagens ?? e.imagens,
      notas_gerais: k.notas_gerais ?? '',
    },
  }
}

export default function FlowDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()

  const flowQuery = useQuery({
    queryKey: ['flow', id],
    queryFn: () => fetchFlow(id!),
    enabled: !!id,
  })

  const agentsQuery = useQuery({
    queryKey: ['agents'],
    queryFn: fetchAgents,
  })

  const form = useForm<FlowEditFormValues>({
    resolver: zodResolver(flowEditFormSchema),
    defaultValues: {
      name: '',
      description: '',
      published: false,
      agent_id: '',
      knowledge: emptyFlowKnowledge(),
    },
  })

  const produtosFA = useFieldArray({ control: form.control, name: 'knowledge.produtos' })
  const servicosFA = useFieldArray({ control: form.control, name: 'knowledge.servicos' })
  const linksFA = useFieldArray({ control: form.control, name: 'knowledge.links' })
  const imagensFA = useFieldArray({ control: form.control, name: 'knowledge.imagens' })
  const slotsFA = useFieldArray({ control: form.control, name: 'knowledge.disponibilidade.slots' })

  useEffect(() => {
    if (flowQuery.data) {
      form.reset(dtoToForm(flowQuery.data))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- reset só quando chegam dados do servidor
  }, [flowQuery.data])

  const saveMut = useMutation({
    mutationFn: async (values: FlowEditFormValues) => {
      const knowledge = sanitizeFlowKnowledge(values.knowledge)
      const body = {
        name: values.name.trim(),
        description: values.description.trim(),
        published: values.published,
        agent_id: values.agent_id.trim(),
        knowledge,
      }
      const res = await api.patch<unknown>(`/flows/${id}`, body)
      return unwrapEnvelope(res).data
    },
    onSuccess: () => {
      toast.success('Fluxo guardado')
      void qc.invalidateQueries({ queryKey: ['flow', id] })
      void qc.invalidateQueries({ queryKey: ['flows'] })
    },
    onError: (e: unknown) => {
      if (e instanceof ApiEnvelopeError) toast.error(e.message)
      else toast.error('Falha ao guardar')
    },
  })

  const delMut = useMutation({
    mutationFn: async () => {
      const res = await api.delete<unknown>(`/flows/${id}`)
      return unwrapEnvelope(res).data
    },
    onSuccess: () => {
      toast.success('Fluxo eliminado')
      void qc.invalidateQueries({ queryKey: ['flows'] })
      navigate('/flows')
    },
    onError: (e: unknown) => {
      if (e instanceof ApiEnvelopeError) toast.error(e.message)
      else toast.error('Falha ao eliminar')
    },
  })

  if (!id) {
    return <p className="p-6 text-text-muted">ID inválido</p>
  }

  if (flowQuery.isLoading) {
    return (
      <div className="p-6 flex items-center gap-2 text-text-muted">
        <Loader2 className="size-5 animate-spin" />
        A carregar fluxo…
      </div>
    )
  }

  if (flowQuery.isError || !flowQuery.data) {
    return (
      <div className="p-6 space-y-4">
        <p className="text-destructive">Não foi possível carregar o fluxo.</p>
        <Button variant="outline" asChild>
          <Link to="/flows">Voltar</Link>
        </Button>
      </div>
    )
  }

  const preview = flowQuery.data.prompt_preview

  return (
    <div className="p-6 max-w-5xl mx-auto flex flex-col gap-6 pb-24">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" asChild>
            <Link to="/flows" aria-label="Voltar">
              <ArrowLeft className="size-5" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-text-primary">Editar fluxo</h1>
            <p className="text-sm text-text-muted">
              Conhecimento usado pelo agente quando o fluxo está <strong>publicado</strong> e ligado ao mesmo agente da auto-resposta WhatsApp.
            </p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            variant="destructive"
            disabled={delMut.isPending}
            onClick={() => {
              if (confirm('Eliminar este fluxo?')) delMut.mutate()
            }}
          >
            Eliminar
          </Button>
          <Button
            form="flow-edit-form"
            type="submit"
            className="bg-primary"
            disabled={saveMut.isPending}
          >
            {saveMut.isPending ? 'A guardar…' : 'Guardar'}
          </Button>
        </div>
      </div>

      <form
        id="flow-edit-form"
        className="space-y-6"
        onSubmit={form.handleSubmit((v) => saveMut.mutate(v))}
      >
        <Card className="bg-card border-border">
          <CardHeader>
            <CardTitle>Informações gerais</CardTitle>
            <CardDescription>Nome, agente e estado de publicação.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4 max-w-xl">
            <div className="space-y-2">
              <Label htmlFor="name">Nome</Label>
              <Input id="name" className="bg-background" {...form.register('name')} />
              {form.formState.errors.name && (
                <p className="text-xs text-destructive">{form.formState.errors.name.message}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Descrição</Label>
              <Textarea id="description" rows={2} className="bg-background" {...form.register('description')} />
            </div>
            <div className="space-y-2">
              <Label>Agente</Label>
              <Select
                value={form.watch('agent_id') || '__none__'}
                onValueChange={(v) => form.setValue('agent_id', v === '__none__' ? '' : v, { shouldDirty: true })}
              >
                <SelectTrigger className="bg-background">
                  <SelectValue placeholder="Nenhum" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">Nenhum</SelectItem>
                  {(agentsQuery.data ?? []).map((a) => (
                    <SelectItem key={a.id} value={a.id}>
                      {a.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-[11px] text-text-muted">
                Para remover o vínculo, escolha «Nenhum» e guarda.
              </p>
            </div>
            <div className="flex items-center justify-between gap-4 rounded-lg border border-border p-3">
              <div>
                <p className="text-sm font-medium">Publicado</p>
                <p className="text-xs text-text-muted">Só fluxos publicados entram na base de conhecimento do modelo.</p>
              </div>
              <Switch
                checked={form.watch('published')}
                onCheckedChange={(v) => form.setValue('published', v, { shouldDirty: true })}
              />
            </div>
          </CardContent>
        </Card>

        <Tabs defaultValue="produtos" className="w-full">
          <TabsList className="flex flex-wrap h-auto gap-1">
            <TabsTrigger value="produtos">Produtos</TabsTrigger>
            <TabsTrigger value="servicos">Serviços</TabsTrigger>
            <TabsTrigger value="horarios">Horários</TabsTrigger>
            <TabsTrigger value="links">Links</TabsTrigger>
            <TabsTrigger value="imagens">Imagens</TabsTrigger>
            <TabsTrigger value="notas">Notas</TabsTrigger>
          </TabsList>

          <TabsContent value="produtos" className="mt-4 space-y-3">
            {produtosFA.fields.map((field, i) => (
              <Card key={field.id} className="bg-sidebar/50 border-border">
                <CardContent className="pt-4 space-y-2">
                  <div className="flex justify-end">
                    <Button type="button" variant="ghost" size="icon" onClick={() => produtosFA.remove(i)}>
                      <Trash2 className="size-4 text-destructive" />
                    </Button>
                  </div>
                  <Input placeholder="Nome" className="bg-background" {...form.register(`knowledge.produtos.${i}.nome`)} />
                  <Textarea placeholder="Descrição" className="bg-background" rows={2} {...form.register(`knowledge.produtos.${i}.descricao`)} />
                  <Input placeholder="Preço referência" className="bg-background" {...form.register(`knowledge.produtos.${i}.preco_referencia`)} />
                </CardContent>
              </Card>
            ))}
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => produtosFA.append({ nome: '', descricao: '', preco_referencia: '' })}
            >
              <Plus className="size-4" />
              Adicionar produto
            </Button>
          </TabsContent>

          <TabsContent value="servicos" className="mt-4 space-y-3">
            {servicosFA.fields.map((field, i) => (
              <Card key={field.id} className="bg-sidebar/50 border-border">
                <CardContent className="pt-4 space-y-2">
                  <div className="flex justify-end">
                    <Button type="button" variant="ghost" size="icon" onClick={() => servicosFA.remove(i)}>
                      <Trash2 className="size-4 text-destructive" />
                    </Button>
                  </div>
                  <Input placeholder="Nome do serviço" className="bg-background" {...form.register(`knowledge.servicos.${i}.nome`)} />
                  <Textarea placeholder="Descrição" className="bg-background" rows={2} {...form.register(`knowledge.servicos.${i}.descricao`)} />
                  <Input placeholder="Duração estimada (ex.: 1h)" className="bg-background" {...form.register(`knowledge.servicos.${i}.duracao_estimada`)} />
                </CardContent>
              </Card>
            ))}
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => servicosFA.append({ nome: '', descricao: '', duracao_estimada: '' })}
            >
              <Plus className="size-4" />
              Adicionar serviço
            </Button>
          </TabsContent>

          <TabsContent value="horarios" className="mt-4 space-y-4">
            <div className="space-y-2">
              <Label>Disponibilidade (texto livre)</Label>
              <Textarea
                rows={4}
                className="bg-background"
                placeholder="Ex.: Segunda a sexta, 9h–18h; sábados sob marcação."
                {...form.register('knowledge.disponibilidade.slots_texto')}
              />
            </div>
            <div className="space-y-2">
              <Label>Observações de horário</Label>
              <Textarea rows={2} className="bg-background" {...form.register('knowledge.disponibilidade.observacoes_horario')} />
            </div>
            <p className="text-xs text-text-muted">Blocos (dia da semana 0=domingo … 6=sábado)</p>
            {slotsFA.fields.map((field, i) => (
              <Card key={field.id} className="bg-sidebar/50 border-border">
                <CardContent className="pt-4 flex flex-wrap gap-2 items-end">
                  <div className="flex-1 min-w-[200px] space-y-1">
                    <Label className="text-xs">Dias (0–6, separados por vírgula)</Label>
                    <Controller
                      control={form.control}
                      name={`knowledge.disponibilidade.slots.${i}.dias_semana`}
                      render={({ field }) => (
                        <Input
                          className="bg-background"
                          placeholder="1,2,3,4,5"
                          value={Array.isArray(field.value) ? field.value.join(',') : ''}
                          onChange={(e) => {
                            const nums = e.target.value
                              .split(/[,\s]+/)
                              .map((x) => parseInt(x.trim(), 10))
                              .filter((n) => !Number.isNaN(n) && n >= 0 && n <= 6)
                            field.onChange(nums)
                          }}
                        />
                      )}
                    />
                  </div>
                  <Input className="w-24 bg-background" placeholder="Início" {...form.register(`knowledge.disponibilidade.slots.${i}.inicio`)} />
                  <Input className="w-24 bg-background" placeholder="Fim" {...form.register(`knowledge.disponibilidade.slots.${i}.fim`)} />
                  <Button type="button" variant="ghost" size="icon" onClick={() => slotsFA.remove(i)}>
                    <Trash2 className="size-4" />
                  </Button>
                </CardContent>
              </Card>
            ))}
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => slotsFA.append({ dias_semana: [1, 2, 3, 4, 5], inicio: '09:00', fim: '18:00' })}
            >
              <Plus className="size-4" />
              Adicionar bloco
            </Button>
          </TabsContent>

          <TabsContent value="links" className="mt-4 space-y-3">
            {linksFA.fields.map((field, i) => (
              <Card key={field.id} className="bg-sidebar/50 border-border">
                <CardContent className="pt-4 flex flex-wrap gap-2">
                  <Input placeholder="Rótulo" className="flex-1 min-w-[120px] bg-background" {...form.register(`knowledge.links.${i}.rotulo`)} />
                  <Input placeholder="https://…" className="flex-[2] min-w-[200px] bg-background" {...form.register(`knowledge.links.${i}.url`)} />
                  <Button type="button" variant="ghost" size="icon" onClick={() => linksFA.remove(i)}>
                    <Trash2 className="size-4" />
                  </Button>
                </CardContent>
              </Card>
            ))}
            <Button type="button" variant="outline" size="sm" onClick={() => linksFA.append({ rotulo: '', url: '' })}>
              <Plus className="size-4" />
              Adicionar link
            </Button>
          </TabsContent>

          <TabsContent value="imagens" className="mt-4 space-y-3">
            <p className="text-xs text-text-muted">URLs públicas (CDN, drive partilhado, etc.). Upload de ficheiros virá numa fase seguinte.</p>
            {imagensFA.fields.map((field, i) => (
              <Card key={field.id} className="bg-sidebar/50 border-border">
                <CardContent className="pt-4 flex flex-wrap gap-2">
                  <Input placeholder="URL da imagem" className="flex-[2] min-w-[200px] bg-background" {...form.register(`knowledge.imagens.${i}.url`)} />
                  <Input placeholder="Legenda" className="flex-1 min-w-[120px] bg-background" {...form.register(`knowledge.imagens.${i}.legenda`)} />
                  <Button type="button" variant="ghost" size="icon" onClick={() => imagensFA.remove(i)}>
                    <Trash2 className="size-4" />
                  </Button>
                </CardContent>
              </Card>
            ))}
            <Button type="button" variant="outline" size="sm" onClick={() => imagensFA.append({ url: '', legenda: '' })}>
              <Plus className="size-4" />
              Adicionar imagem
            </Button>
          </TabsContent>

          <TabsContent value="notas" className="mt-4 space-y-2">
            <Label>Notas gerais / FAQ / políticas</Label>
            <Textarea rows={12} className="bg-background font-mono text-sm" {...form.register('knowledge.notas_gerais')} />
          </TabsContent>
        </Tabs>
      </form>

      <Card className="bg-card border-border">
        <CardHeader>
          <CardTitle className="text-base">Pré-visualização enviada ao modelo</CardTitle>
          <CardDescription>
            Texto derivado deste fluxo (o servidor pode truncar o total se houver vários fluxos publicados).
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ScrollArea className="h-[min(40vh,320px)] rounded-md border border-border p-3 bg-muted/30">
            <pre className="text-xs whitespace-pre-wrap text-text-muted font-sans">
              {preview || '— Sem conteúdo estruturado —'}
            </pre>
          </ScrollArea>
        </CardContent>
      </Card>
    </div>
  )
}
