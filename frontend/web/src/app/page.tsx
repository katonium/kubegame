import { KubeWarsGame } from '@/components/game/KubeWarsGame';

export default function Home() {
  return (
    <main className="h-screen bg-background text-foreground overflow-hidden">
      <KubeWarsGame />
    </main>
  );
}
