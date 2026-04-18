import { useQuery } from '@tanstack/react-query'
import { GitBranch, Plus } from 'lucide-react'
import { api, unwrapEnvelope } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'

type FlowListItem = {
  id: string
  name: string
  description: string
  agentName: string
  published: boolean
}

async function fetchFlows(): Promise<FlowListItem[]> {
  const res = await api.get<unknown>('/flows')
  const { data } = unwrapEnvelope<FlowListItem[]>(res)
  return data
}

export default function Flows() {
  const { data = [] } = useQuery({
    queryKey: ['flows'],
    queryFn: fetchFlows,
  })

  return (
    <div className="p-6 h-full flex flex-col gap-4 min-h-0">
      <div className="flex items-center justify-between shrink-0">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Fluxos</h1>
          <p className="text-sm text-text-muted">Motor de fluxos — lista vinda da API (placeholder)</p>
        </div>
        <Button className="bg-primary">
          <Plus className="size-4" />
          Novo fluxo
        </Button>
      </div>

      <div className="grid lg:grid-cols-2 gap-4 flex-1 min-h-0">
        <ScrollArea className="rounded-xl border border-border bg-card h-[min(60vh,480px)]">
          <div className="p-4 space-y-3">
            {data.map((f) => (
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
                  {f.description} · {f.agentName}
                </CardContent>
              </Card>
            ))}
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
            Canvas manual — conecte nós com PATCH /api/v1/flows/:id
          </p>
        </div>
      </div>
    </div>
  )
}
