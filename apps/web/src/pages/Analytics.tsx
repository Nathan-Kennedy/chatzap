import { useQuery } from '@tanstack/react-query'
import { api, unwrapEnvelope } from '@/lib/api'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
} from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

const lineData = [
  { day: 'Seg', abertas: 12, resolvidas: 8 },
  { day: 'Ter', abertas: 19, resolvidas: 14 },
  { day: 'Qua', abertas: 15, resolvidas: 15 },
  { day: 'Qui', abertas: 22, resolvidas: 18 },
  { day: 'Sex', abertas: 18, resolvidas: 20 },
]

const barData = [
  { h: '9h', vol: 40 },
  { h: '12h', vol: 85 },
  { h: '15h', vol: 55 },
  { h: '18h', vol: 70 },
]

type Overview = {
  messages_last_30d: number
  conversations_total: number
  instances_total: number
}

async function fetchOverview(): Promise<Overview> {
  const res = await api.get<unknown>('/analytics/overview')
  const { data } = unwrapEnvelope<Overview>(res)
  return data
}

export default function Analytics() {
  const { data: overview } = useQuery({
    queryKey: ['analytics', 'overview'],
    queryFn: fetchOverview,
  })

  return (
    <div className="p-6 h-full overflow-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-text-primary">Analytics</h1>
        <p className="text-sm text-text-muted">Resumo do workspace (últimos 30 dias — mensagens)</p>
      </div>

      <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {[
          { t: 'Mensagens (30d)', v: String(overview?.messages_last_30d ?? '—') },
          { t: 'Conversas', v: String(overview?.conversations_total ?? '—') },
          { t: 'Instâncias', v: String(overview?.instances_total ?? '—') },
        ].map((k) => (
          <Card key={k.t} className="bg-card border-border">
            <CardHeader className="pb-1">
              <CardTitle className="text-sm font-medium text-text-muted">
                {k.t}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl font-bold text-text-primary">{k.v}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid lg:grid-cols-2 gap-6">
        <Card className="bg-card border-border">
          <CardHeader>
            <CardTitle className="text-base">Abertas vs resolvidas</CardTitle>
          </CardHeader>
          <CardContent className="h-[260px]">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={lineData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                <XAxis dataKey="day" className="text-xs" />
                <YAxis className="text-xs" />
                <Tooltip
                  contentStyle={{
                    background: '#15151F',
                    border: '1px solid rgba(255,255,255,0.08)',
                  }}
                />
                <Line type="monotone" dataKey="abertas" stroke="#7C3AED" name="Abertas" />
                <Line type="monotone" dataKey="resolvidas" stroke="#10B981" name="Resolvidas" />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        <Card className="bg-card border-border">
          <CardHeader>
            <CardTitle className="text-base">Volume por horário</CardTitle>
          </CardHeader>
          <CardContent className="h-[260px]">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={barData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                <XAxis dataKey="h" />
                <YAxis />
                <Tooltip
                  contentStyle={{
                    background: '#15151F',
                    border: '1px solid rgba(255,255,255,0.08)',
                  }}
                />
                <Bar dataKey="vol" fill="#06B6D4" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
