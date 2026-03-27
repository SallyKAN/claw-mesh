import { MeshEvent } from './types'

export function connectWS(onEvent: (event: MeshEvent) => void): () => void {
  return () => {}
}
