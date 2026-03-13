'use client'

interface WizardStepperProps {
  currentStep: number
  totalSteps: number
}

export function WizardStepper({ currentStep, totalSteps }: WizardStepperProps) {
  return (
    <div className="flex items-center gap-3">
      <span className="text-xs font-bold uppercase tracking-widest text-text-dim">
        STEP {currentStep}/{totalSteps}
      </span>
      <div className="flex items-center gap-1.5">
        {Array.from({ length: totalSteps }, (_, i) => {
          const step = i + 1
          const isCompleted = step < currentStep
          const isCurrent = step === currentStep

          return (
            <div
              key={step}
              className={`h-2 w-2 rounded-full transition-colors ${
                isCompleted
                  ? 'bg-green'
                  : isCurrent
                    ? 'bg-amber'
                    : 'bg-muted'
              }`}
            />
          )
        })}
      </div>
    </div>
  )
}
