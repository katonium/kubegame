# **App Name**: KubeWars

## Core Features:

- Panel Display: Display a split screen with player and Kubernetes scheduler panels.
- Pod Representation: Visually represent Pods with labels (Banana, Chocolate, etc.) and requirements at the top of the panel.
- Drag-and-Drop Scheduling and Click to Schedling: Allow players to drag and drop Pods onto Nodes to simulate scheduling. Click pod and then click node to schedule is also supported.
- Status Updates: Update Pod status from 'Pending' to 'Scheduling' to 'Running' based on successful placement. Status will be changed 1 sec after plaement.
- Scoring System: Implement a scoring system where players earn points per second for correctly scheduled and running Pods, taking requirements into account. Stop awarding point if a Pod is terminated.
- In-Memory Game Engine: Implement a simple in-memory game engine to manage Pod termination and Node creation/termination. This will eventually be replaced with a Golang backend.
- Timer and Score Display: Display a game timer and score in the right-top panel.
- Initial Start/Help Panel: When user open web page, user get panel that have "start"  and "how to start". When user click how to start, the panel changed view to show how to play game. <- is placed left-top of the panel, user clicked to go back home.

## Style Guidelines:

- Primary color: Blue (#326CE5) to capture the high-stakes atmosphere.
- Background color: White (#FFFFFF) to ensure contrast and readability.
- Accent color: Electric cyan (#7DF9FF) for highlighting interactive elements and status changes.
- Font: 'Inter', a grotesque-style sans-serif font with a neutral, modern feel suitable for the user interface elements, status indicators, and in-game text.
- Use simple, clear icons to represent Pod labels and requirements.
- Maintain a clear separation between the player's panel and the Kubernetes scheduler panel using visual cues such as borders or background variations.
- Use subtle animations to indicate status changes (Pending, Scheduling, Running) and when a player earns points.