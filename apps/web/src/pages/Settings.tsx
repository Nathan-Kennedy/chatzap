import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { toast } from 'sonner'
import { useAuthProfile } from '@/hooks/useAuthProfile'
import { api, unwrapEnvelope } from '@/lib/api'
import { ApiEnvelopeError } from '@/types/api'

const profileSchema = z.object({
  name: z.string().min(2),
  email: z.string().email(),
})

type ProfileForm = z.infer<typeof profileSchema>

const workspaceSchema = z.object({
  name: z.string().min(2),
})

type WorkspaceForm = z.infer<typeof workspaceSchema>

async function getWorkspace() {
  const res = await api.get<unknown>('/workspace')
  return unwrapEnvelope<{ id: string; name: string }>(res).data
}

export default function Settings() {
  const profile = useAuthProfile()
  const qc = useQueryClient()

  const form = useForm<ProfileForm>({
    resolver: zodResolver(profileSchema),
    defaultValues: { name: '', email: '' },
  })

  const wsForm = useForm<WorkspaceForm>({
    resolver: zodResolver(workspaceSchema),
    defaultValues: { name: '' },
  })

  useEffect(() => {
    if (profile) {
      form.reset({
        name: profile.user_name ?? '',
        email: profile.user_email ?? '',
      })
    }
  }, [profile, form])

  const { data: ws } = useQuery({
    queryKey: ['workspace'],
    queryFn: getWorkspace,
  })

  useEffect(() => {
    if (ws?.name) wsForm.reset({ name: ws.name })
  }, [ws, wsForm])

  const patchWs = useMutation({
    mutationFn: async (name: string) => {
      const res = await api.patch('/workspace', { name })
      return unwrapEnvelope<{ id: string; name: string }>(res).data
    },
    onSuccess: () => {
      toast.success('Workspace atualizado')
      void qc.invalidateQueries({ queryKey: ['workspace'] })
    },
    onError: (e: unknown) => {
      if (e instanceof ApiEnvelopeError) toast.error(e.message)
      else toast.error('Falha ao guardar')
    },
  })

  return (
    <div className="p-6 h-full overflow-auto">
      <h1 className="text-2xl font-bold text-text-primary mb-6">Configurações</h1>

      <Tabs defaultValue="conta" className="max-w-3xl">
        <TabsList className="bg-card border border-border flex flex-wrap h-auto gap-1 p-1">
          <TabsTrigger value="conta">Conta</TabsTrigger>
          <TabsTrigger value="workspace">Workspace</TabsTrigger>
          <TabsTrigger value="integracoes">Integrações</TabsTrigger>
          <TabsTrigger value="api">API Keys</TabsTrigger>
          <TabsTrigger value="notif">Notificações</TabsTrigger>
        </TabsList>

        <TabsContent value="conta" className="mt-4">
          <Card className="bg-card border-border">
            <CardHeader>
              <CardTitle>Perfil</CardTitle>
              <CardDescription>Dados da sessão atual (edição de perfil no servidor em breve)</CardDescription>
            </CardHeader>
            <CardContent>
              <form
                className="space-y-4 max-w-md"
                onSubmit={form.handleSubmit(() =>
                  toast.info('Atualização de perfil: em breve (PATCH /users/me)')
                )}
              >
                <div className="space-y-2">
                  <Label htmlFor="s-name">Nome</Label>
                  <Input id="s-name" {...form.register('name')} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="s-email">E-mail</Label>
                  <Input id="s-email" type="email" {...form.register('email')} />
                </div>
                <Button type="submit">Salvar</Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="workspace" className="mt-4">
          <Card className="bg-card border-border">
            <CardHeader>
              <CardTitle>Workspace</CardTitle>
              <CardDescription>Nome visível no painel</CardDescription>
            </CardHeader>
            <CardContent>
              <form
                className="space-y-4 max-w-md"
                onSubmit={wsForm.handleSubmit((v) => patchWs.mutate(v.name))}
              >
                <div className="space-y-2">
                  <Label htmlFor="ws-name">Nome do workspace</Label>
                  <Input id="ws-name" {...wsForm.register('name')} />
                </div>
                <Button type="submit" disabled={patchWs.isPending}>
                  {patchWs.isPending ? 'A guardar…' : 'Guardar'}
                </Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="integracoes" className="mt-4">
          <Card className="bg-card border-border">
            <CardHeader>
              <CardTitle>Integrações</CardTitle>
              <CardDescription>Evolution API, LLMs — secrets só no backend</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between rounded-lg border border-border p-4">
                <span className="text-sm">Evolution API</span>
                <Switch disabled />
              </div>
              <div className="flex items-center justify-between rounded-lg border border-border p-4">
                <span className="text-sm">OpenAI</span>
                <Switch disabled />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="api" className="mt-4">
          <Card className="bg-card border-border">
            <CardHeader>
              <CardTitle>Chaves de API</CardTitle>
              <CardDescription>Geridas no servidor (futuro)</CardDescription>
            </CardHeader>
            <CardContent>
              <Button variant="secondary" size="sm" disabled>
                Gerar nova chave
              </Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="notif" className="mt-4">
          <Card className="bg-card border-border">
            <CardHeader>
              <CardTitle>Notificações</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {['Nova conversa', 'Mensagem não lida', 'Atribuído a mim'].map((n) => (
                <div key={n} className="flex items-center justify-between">
                  <span className="text-sm">{n}</span>
                  <Switch defaultChecked />
                </div>
              ))}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
