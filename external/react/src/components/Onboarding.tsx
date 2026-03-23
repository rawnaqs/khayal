import { useState } from 'react'
import { motion } from 'framer-motion'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

interface OnboardingProps {
  onComplete: () => void
}

export function Onboarding({ onComplete }: OnboardingProps) {
  const [token, setToken] = useState('')
  const [error, setError] = useState('')
  const [testing, setTesting] = useState(false)

  const testConnection = async () => {
    if (!token) {
      setError('Please enter your token')
      return
    }

    setTesting(true)
    setError('')

    try {
      const host = window.location.origin
      const response = await fetch(`${host}/v1/health`, {
        headers: { 'X-Khayal-Token': token },
      })

      if (!response.ok) {
        throw new Error('Invalid token')
      }

      localStorage.setItem('khayal_host', host)
      localStorage.setItem('khayal_token', token)
      onComplete()
    } catch {
      setError('Cannot connect. Check your token.')
    } finally {
      setTesting(false)
    }
  }

  return (
    <div className="flex flex-col items-center justify-center h-screen p-6 bg-background">
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.4, ease: "easeOut" }}
      >
        <Card className="w-full max-w-sm glass border-primary/20 shadow-[0_0_40px_hsl(var(--primary)/0.1)]">
          <CardHeader className="text-center pb-2">
            <motion.div
              initial={{ scale: 0.8, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              transition={{ delay: 0.1, duration: 0.4, ease: "easeOut" }}
              className="w-16 h-16 mx-auto mb-4 rounded-2xl flex items-center justify-center"
            >
              <img src="/icon.svg" alt="khayal" className="w-16 h-16" />
            </motion.div>
            <CardTitle className="text-2xl font-bold tracking-tight">khayal</CardTitle>
            <p className="text-caption mt-1">private second brain</p>
          </CardHeader>
          <CardContent className="space-y-4 pt-4">
            <Input
              placeholder="token"
              type="password"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  testConnection()
                }
              }}
              className="glass input-glow transition-all duration-300"
            />
            {error && (
              <motion.p
                initial={{ opacity: 0, y: -4 }}
                animate={{ opacity: 1, y: 0 }}
                className="text-sm text-destructive text-center"
              >
                {error}
              </motion.p>
            )}
            <motion.div whileTap={{ scale: 0.98 }}>
              <Button
                className="w-full h-12 btn-gradient font-semibold tracking-wide"
                onClick={testConnection}
                disabled={testing}
              >
                {testing ? (
                  <span className="animate-pulse">connecting...</span>
                ) : (
                  'connect'
                )}
              </Button>
            </motion.div>
          </CardContent>
        </Card>
      </motion.div>
    </div>
  )
}
