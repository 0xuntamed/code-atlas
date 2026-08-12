// Shared layout + density constants. GraphCanvas and the ELK layout worker both
// import these so the node box the worker reserves can never drift from the card
// AtlasNode actually renders.
export const NODE_WIDTH = 214
export const NODE_HEIGHT = 82

export const DENSITY = {
  // Above this node count, cards render compact and edge labels/animation drop.
  compactAbove: 60,
  // Edge labels are only drawn at or below this node count.
  edgeLabelsUpTo: 30,
  // The MiniMap only appears at or below this node count.
  minimapUpTo: 70,
  // Edges animate only below this node count.
  animateBelow: 45,
}
