import { Link, useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { MessageSquare } from 'lucide-react'
import { toast } from 'sonner'
import { loginRequest } from '@/lib/auth-api'
import { setAuthProfile } from '@/lib/auth-storage'
import { ApiEnvelopeError } from '@/types/api'
import { authProfileQueryKey } from '@/hooks/useAuthProfile'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const loginSchema = z.object({
  email: z
    .string()
    .min(1, 'E-mail obrigatório')
    .email('E-mail inválido'),
  password: z.string().min(1, 'Senha obrigatória'),
})

type LoginForm = z.infer<typeof loginSchema>

export default function Login() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const form = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: '', password: '' },
  })

  const mutation = useMutation({
    mutationFn: loginRequest,
    onSuccess: (data) => {
      setAuthProfile({
        workspace_id: data.workspace_id,
        workspace_name: data.workspace_name,
        role: data.user.role,
        user_name: data.user.name,
        user_email: data.user.email,
      })
      void queryClient.invalidateQueries({ queryKey: authProfileQueryKey })
      toast.success('Login realizado')
      navigate('/inbox', { replace: true })
    },
    onError: (err: unknown) => {
      if (err instanceof ApiEnvelopeError) {
        toast.error(err.message)
        return
      }
      toast.error('Não foi possível entrar. Verifique a API e tente novamente.')
    },
  })

  const privacyUrl =
    import.meta.env.VITE_PRIVACY_URL ?? 'https://example.com/privacidade'
  const termsUrl =
    import.meta.env.VITE_TERMS_URL ?? 'https://example.com/termos'

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4 relative overflow-hidden">
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] bg-primary/20 blur-[120px] rounded-full pointer-events-none" />

      <div className="w-full max-w-sm bg-card border border-border rounded-xl shadow-2xl p-8 relative z-10">
        <div className="flex flex-col items-center mb-8">
          <div className="w-12 h-12 bg-primary rounded-xl flex items-center justify-center mb-4 shadow-lg shadow-primary/20">
            <MessageSquare className="text-white" size={24} />
          </div>
          <h1 className="text-2xl font-bold text-text-primary">WhatsSaaS</h1>
          <p className="text-sm text-text-muted mt-1">Faça login na sua conta</p>
        </div>

        <form
          onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
          className="space-y-4"
        >
          <div className="space-y-1.5">
            <Label htmlFor="email" className="text-text-secondary">
              E-mail
            </Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              placeholder="seu@email.com"
              className="bg-background border-border"
              {...form.register('email')}
            />
            {form.formState.errors.email && (
              <p className="text-xs text-destructive">
                {form.formState.errors.email.message}
              </p>
            )}
          </div>
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <Label htmlFor="password" className="text-text-secondary">
                Senha
              </Label>
              <span className="text-xs text-text-muted">Esqueceu? (em breve)</span>
            </div>
            <Input
              id="password"
              type="password"
              autoComplete="current-password"
              placeholder="••••••••"
              className="bg-background border-border"
              {...form.register('password')}
            />
            {form.formState.errors.password && (
              <p className="text-xs text-destructive">
                {form.formState.errors.password.message}
              </p>
            )}
          </div>
          <Button
            type="submit"
            disabled={mutation.isPending}
            className="w-full mt-4 bg-primary hover:bg-primary-hover"
          >
            {mutation.isPending ? 'Entrando…' : 'Entrar'}
          </Button>
        </form>

        <p className="text-sm text-center text-text-muted mt-4">
          Não tem conta?{' '}
          <Link to="/register" className="text-primary hover:underline">
            Criar conta
          </Link>
        </p>

        <p className="text-[11px] text-text-muted text-center mt-6 leading-relaxed">
          Ao continuar, você concorda com os{' '}
          <a
            href={termsUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary hover:underline"
          >
            Termos
          </a>{' '}
          e a{' '}
          <a
            href={privacyUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary hover:underline"
          >
            Política de Privacidade
          </a>
          .
        </p>
      </div>
    </div>
  )
}
