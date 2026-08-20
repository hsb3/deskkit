<script lang="ts">
  // One <svg> for every glyph; the paths live in icons.ts. Always aria-hidden: an icon here
  // decorates a control that already carries its own word or digit.
  import { ICONS, type IconName } from './icons'

  let { name, size = 14 }: { name: IconName; size?: number } = $props()
</script>

<svg
  viewBox="0 0 24 24"
  width={size}
  height={size}
  fill="none"
  stroke="currentColor"
  stroke-width="1.7"
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
  focusable="false"><path d={ICONS[name]} /></svg
>

<style>
  svg {
    flex: 0 0 auto;
    display: block;
  }
</style>
