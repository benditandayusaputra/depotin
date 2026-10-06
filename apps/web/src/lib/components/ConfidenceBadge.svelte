<script lang="ts">
  import Badge, { type Tone } from './Badge.svelte';
  import { confidenceLabel, type Confidence } from '$lib/utils/labels';

  let { confidence }: { confidence: Confidence } = $props();

  const tones: Record<Confidence, Tone> = {
    none: 'neutral',
    low: 'pending',
    medium: 'confirmed',
    high: 'done'
  };
</script>

<Badge tone={tones[confidence]}>Keyakinan {confidenceLabel(confidence).toLowerCase()}</Badge>
