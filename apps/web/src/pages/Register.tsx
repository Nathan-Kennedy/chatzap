import { Link, useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { MessageSquare } from 'lucide-react'
import { toast } from 'sonner'
import { registerRequest } from '@/lib/auth-api'
import { setAuthProfile } from '@/lib/auth-storage'
import { ApiEnvelopeError } from '@/types/api'
import { authProfileQueryKey } from '@/hooks/useAuthProfile'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const schema = z.object({
  name: z.string().min(2, 'Nome obrigatório'),
  email: z.string().email('E-mail inválido'),
  password: z.string().min(8, 'Mínimo 8 caracteres'),
  workspace_name: z.string().min(2, 'Nome do workspace'),
})

type Form = z.infer<typeof schema>

export default function Register() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const form = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      email: '',
      password: '',
      workspace_name: 'Meu workspace',
    },
  })

  const mutation = useMutation({
    mutationFn: registerRequest,
    onSuccess: (data) => {
      setAuthProfile({
        workspace_id: data.workspace_id,
        workspace_name: data.workspace_name,
        role: data.user.role,
        user_name: data.user.name,
        user_email: data.user.email,
      })
      void qc.invalidateQueries({ queryKey: authProfileQueryKey })
      toast.success('Conta criada')
      navigate('/inbox', { replace: true })
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) toast.error(err.message)
      else toast.error('Não foi possível registar.')
    },
  })

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <div className="w-full max-w-sm bg-card border border-border rounded-xl shadow-2xl p-8">
        <div className="flex flex-col items-center mb-6">
          <div className="w-12 h-12 bg-primary rounded-xl flex items-center justify-center mb-3">
            <MessageSquare className="text-white" size={24} />
          </div>
          <h1 className="text-2xl font-bold text-text-primary">Criar conta</h1>
          <p className="text-sm text-text-muted mt-1">Workspace + primeiro utilizador</p>
        </div>

        <form
          onSubmit={form.handleSubmit((v) => mutation.mutate(v))}
          className="space-y-3"
        >
          <div className="space-y-1">
            <Label htmlFor="ws">Workspace</Label>
            <Input id="ws" {...form.register('workspace_name')} className="bg-background" />
            {form.formState.errors.workspace_name && (
              <p className="text-xs text-destructive">
                {form.formState.errors.workspace_name.message}
              </p>
            )}
          </div>
          <div className="space-y-1">
            <Label htmlFor="name">O teu nome</Label>
            <Input id="name" {...form.register('name')} className="bg-background" />
            {form.formState.errors.name && (
              <p className="text-xs text-destructive">{form.formState.errors.name.message}</p>
            )}
          </div>
          <div className="space-y-1">
            <Label htmlFor="email">E-mail</Label>
            <Input id="email" type="email" {...form.register('email')} className="bg-background" />
            {form.formState.errors.email && (
              <p className="text-xs text-destructive">{form.formState.errors.email.message}</p>
            )}
          </div>
          <div className="space-y-1">
            <Label htmlFor="pw">Senha</Label>
            <Input id="pw" type="password" {...form.register('password')} className="bg-background" />
            {form.formState.errors.password && (
              <p className="text-xs text-destructive">{form.formState.errors.password.message}</p>
            )}
          </div>
          <Button type="submit" className="w-full mt-2" disabled={mutation.isPending}>
            {mutation.isPending ? 'A criar…' : 'Registar'}
          </Button>
        </form>

        <p className="text-sm text-center text-text-muted mt-6">
          Já tens conta?{' '}
          <Link to="/login" className="text-primary hover:underline">
            Entrar
          </Link>
        </p>
      </div>
    </div>
  )
}
