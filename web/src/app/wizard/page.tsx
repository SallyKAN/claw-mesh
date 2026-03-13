'use client'

import { useState } from 'react'
import { WizardStepper } from '@/components/wizard/wizard-stepper'
import { NetworkStep } from '@/components/wizard/network-step'
import { ConfigStep } from '@/components/wizard/config-step'
import { ExecStep } from '@/components/wizard/exec-step'

interface WizardData {
  network: { type: 'lan' | 'public'; coordinatorUrl: string }
  config: { name: string; tags: string }
}

const TOTAL_STEPS = 3

export default function WizardPage() {
  const [step, setStep] = useState(1)
  const [data, setData] = useState<WizardData>({
    network: { type: 'lan', coordinatorUrl: '' },
    config: { name: '', tags: '' },
  })

  return (
    <div className="space-y-8 max-w-2xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-bold uppercase tracking-widest text-green">
          ADD NODE TO MESH
        </h1>
        <WizardStepper currentStep={step} totalSteps={TOTAL_STEPS} />
      </div>

      {/* Step content */}
      <div className="border border-border bg-surface p-6">
        {step === 1 && (
          <NetworkStep
            value={data.network}
            onChange={(network) => setData((d) => ({ ...d, network }))}
          />
        )}
        {step === 2 && (
          <ConfigStep
            value={data.config}
            onChange={(config) => setData((d) => ({ ...d, config }))}
          />
        )}
        {step === 3 && (
          <ExecStep
            config={{
              networkType: data.network.type,
              coordinatorUrl: data.network.coordinatorUrl,
              name: data.config.name,
              tags: data.config.tags,
            }}
          />
        )}
      </div>

      {/* Navigation */}
      <div className="flex items-center justify-between">
        <button
          type="button"
          onClick={() => setStep((s) => s - 1)}
          disabled={step === 1}
          className="border border-border text-text-dim bg-transparent hover:border-border-bright hover:text-text px-4 py-2 text-xs font-bold uppercase tracking-widest transition-colors disabled:opacity-50 disabled:pointer-events-none"
        >
          BACK
        </button>
        <button
          type="button"
          onClick={() => setStep((s) => s + 1)}
          disabled={step === TOTAL_STEPS}
          className="border border-green text-green bg-transparent hover:bg-green hover:text-bg px-4 py-2 text-xs font-bold uppercase tracking-widest transition-colors disabled:opacity-50 disabled:pointer-events-none"
        >
          {step === TOTAL_STEPS ? 'DONE' : 'NEXT →'}
        </button>
      </div>
    </div>
  )
}
