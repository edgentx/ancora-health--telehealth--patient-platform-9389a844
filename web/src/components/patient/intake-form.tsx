'use client';

import { useState } from 'react';

import { useSubmitIntake, type IntakeAnswer } from '@/lib/api';
import { useUiStore } from '@/store/ui-store';

/**
 * Pre-visit intake. The questionnaire is a fixed set of fields; on submit we
 * pack them into the {@link IntakeAnswer} shape the SubmitIntakeCmd expects. A
 * successful submit swaps the form for a confirmation, mirroring the aggregate's
 * pending → submitted transition.
 */
const QUESTIONS: ReadonlyArray<{ key: string; label: string; multiline?: boolean }> = [
  { key: 'chief_complaint', label: 'What brings you in today?', multiline: true },
  { key: 'symptom_duration', label: 'How long have you had these symptoms?' },
  { key: 'allergies', label: 'List any drug or food allergies' },
  { key: 'current_medications', label: 'Current medications', multiline: true },
];

export function IntakeForm() {
  const submit = useSubmitIntake();
  const patientId = useUiStore((s) => s.user) ?? 'self';
  const [values, setValues] = useState<Record<string, string>>({});

  function onSubmit(event: React.FormEvent) {
    event.preventDefault();
    const answers: IntakeAnswer[] = QUESTIONS.map((q) => ({
      key: q.key,
      label: q.label,
      value: values[q.key]?.trim() ?? '',
    }));
    submit.mutate({ patientId, answers });
  }

  if (submit.isSuccess) {
    return (
      <section>
        <h1 className="page-heading">Intake submitted</h1>
        <p className="banner banner--success" role="status">
          Thanks — your care team has your pre-visit information. You can update it
          any time before your appointment.
        </p>
      </section>
    );
  }

  return (
    <section>
      <h1 className="page-heading">Pre-visit intake</h1>
      <p className="page-subheading">
        A few questions so your provider is ready before the visit.
      </p>

      <form className="card" onSubmit={onSubmit} aria-label="Intake form">
        {QUESTIONS.map((q) => (
          <div className="field" key={q.key}>
            <label className="field__label" htmlFor={`intake-${q.key}`}>
              {q.label}
            </label>
            {q.multiline ? (
              <textarea
                id={`intake-${q.key}`}
                className="textarea"
                value={values[q.key] ?? ''}
                onChange={(e) => setValues((v) => ({ ...v, [q.key]: e.target.value }))}
              />
            ) : (
              <input
                id={`intake-${q.key}`}
                className="input"
                value={values[q.key] ?? ''}
                onChange={(e) => setValues((v) => ({ ...v, [q.key]: e.target.value }))}
              />
            )}
          </div>
        ))}

        {submit.isError ? (
          <p className="banner banner--conflict" role="alert">
            We couldn’t submit your intake. Please try again.
          </p>
        ) : null}

        <div className="form-actions">
          <button type="submit" className="btn btn--primary" disabled={submit.isPending}>
            {submit.isPending ? 'Submitting…' : 'Submit intake'}
          </button>
        </div>
      </form>
    </section>
  );
}
