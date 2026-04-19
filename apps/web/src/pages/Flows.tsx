import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { GitBranch, Plus } from 'lucide-react'
import { api, unwrapEnvelope } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toast } from 'sonner'
import { ApiEnvelopeError } from '@/types/api'

type FlowListItem = {
  id: string
  name: string
  description: string
  agentName: string
  published: boolean
}

type AgentListRow = {
  id: string
  name: string
}

async function fetchFlows(): Promise<FlowListItem[]> {
  const res = await api.get<unknown>('/flows')
  const { data } = unwrapEnvelope<FlowListItem[]>(res)
  return data
}

async function fetchAgents(): Promise<AgentListRow[]> {
  const res = await api.get<unknown>('/agents')
  const { data } = unwrapEnvelope<AgentListRow[]>(res)
  return data
}

export default function Flows() {
  const qc = useQueryClient()
  const [sheetOpen, setSheetOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [agentId, setAgentId] = useState<string>('')

  const { data = [], isLoading } = useQuery({
    queryKey: ['flows'],
    queryFn: fetchFlows,
  })

  const { data: agents = [] } = useQuery({
    queryKey: ['agents'],
    queryFn: fetchAgents,
  })

  const createMut = useMutation({
    mutationFn: async () => {
      const body: { name: string; description: string; agent_id?: string } = {
        name: name.trim(),
        description: description.trim(),
      }
      if (agentId.trim()) body.agent_id = agentId.trim()
      const res = await api.post<unknown>('/flows', body)
      return unwrapEnvelope<{ id: string }>(res).data
    },
    onSuccess: () => {
      toast.success('Fluxo criado')
      setSheetOpen(false)
      setName('')
      setDescription('')
      setAgentId('')
      void qc.invalidateQueries({ queryKey: ['flows'] })
    },
    onError: (e: unknown) => {
      if (e instanceof ApiEnvelopeError) toast.error(e.message)
      else toast.error('Falha ao criar fluxo')
    },
  })

  return (
    <div className="p-6 h-full flex flex-col gap-4 min-h-0">
      <div className="flex items-center justify-between shrink-0">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Fluxos</h1>
          <p className="text-sm text-text-muted">
            Modelos guardados no servidor (MVP). Execução automática e editor de nós ficam para fases seguintes.
          </p>
        </div>
        <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
          <SheetTrigger asChild>
            <Button className="bg-primary">
              <Plus className="size-4" />
              Novo fluxo
            </Button>
          </SheetTrigger>
          <SheetContent className="bg-card border-border w-full sm:max-w-md">
            <SheetHeader>
              <SheetTitle>Novo fluxo</SheetTitle>
            </SheetHeader>
            <div className="mt-6 space-y-4">
              <div className="space-y-2">
                <Label htmlFor="flow-name">Nome</Label>
                <Input id="flow-name" value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="flow-desc">Descrição</Label>
                <Textarea id="flow-desc" value={description} onChange={(e) => setDescription(e.target.value)} rows={3} />
              </div>
              <div className="space-y-2">
                <Label>Agente (opcional)</Label>
                <Select value={agentId || '__none__'} onValueChange={(v) => setAgentId(v === '__none__' ? '' : v)}>
                  <SelectTrigger className="bg-background">
                    <SelectValue placeholder="Nenhum" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__none__">Nenhum</SelectItem>
                    {agents.map((a) => (
                      <SelectItem key={a.id} value={a.id}>
                        {a.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <Button
                className="w-full"
                disabled={createMut.isPending || !name.trim()}
                onClick={() => createMut.mutate()}
              >
                {createMut.isPending ? 'A criar…' : 'Criar fluxo'}
              </Button>
            </div>
          </SheetContent>
        </Sheet>
      </div>

      <div className="grid lg:grid-cols-2 gap-4 flex-1 min-h-0">
        <ScrollArea className="rounded-xl border border-border bg-card h-[min(60vh,480px)]">
          <div className="p-4 space-y-3">
            {isLoading ? (
              <p className="text-sm text-text-muted">A carregar…</p>
            ) : data.length === 0 ? (
              <p className="text-sm text-text-muted">Nenhum fluxo ainda. Cria um com &quot;Novo fluxo&quot;.</p>
            ) : (
              data.map((f) => (
                <Card key={f.id} className="bg-sidebar/80 border-border">
                  <CardHeader className="py-3">
                    <div className="flex items-center justify-between gap-2">
                      <CardTitle className="text-base flex items-center gap-2">
                        <GitBranch className="size-4 text-primary" />
                        {f.name}
                      </CardTitle>
                      <Badge variant={f.published ? 'default' : 'secondary'}>
                        {f.published ? 'Publicado' : 'Rascunho'}
                      </Badge>
                    </div>
                  </CardHeader>
                  <CardContent className="text-xs text-text-muted pb-3">
                    {f.description || '—'}
                    {f.agentName ? ` · Agente: ${f.agentName}` : ''}
                  </CardContent>
                </Card>
              ))
            )}
          </div>
        </ScrollArea>

        <div className="rounded-xl border border-border bg-card relative overflow-hidden min-h-[280px]">
          <div className="absolute inset-0 flex items-center justify-center p-6">
            <svg className="w-full h-full max-w-md" viewBox="0 0 400 200">
              <path
                d="M 80 100 C 140 100 160 40 200 40 C 240 40 260 100 320 100"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                className="text-primary/50"
              />
              <rect
                x="20"
                y="70"
                width="60"
                height="60"
                rx="8"
                className="fill-card stroke-primary stroke-2"
              />
              <text x="50" y="105" textAnchor="middle" className="fill-text-primary text-[10px]">
                Início
              </text>
              <rect
                x="170"
                y="10"
                width="60"
                height="60"
                rx="8"
                className="fill-card stroke-violet-500 stroke-2"
              />
              <text x="200" y="45" textAnchor="middle" className="fill-text-primary text-[10px]">
                Msg
              </text>
              <rect
                x="320"
                y="70"
                width="60"
                height="60"
                rx="8"
                className="fill-card stroke-success stroke-2"
              />
              <text x="350" y="105" textAnchor="middle" className="fill-text-primary text-[10px]">
                Fim
              </text>
            </svg>
          </div>
          <p className="absolute bottom-3 left-0 right-0 text-center text-[11px] text-text-muted">
            Pré-visualização ilustrativa — editor de fluxo em breve.
          </p>
        </div>
      </div>
    </div>
  )
}
