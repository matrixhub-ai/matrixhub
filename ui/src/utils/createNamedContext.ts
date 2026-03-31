import { createContext } from 'react'

export const createNamedContext = <T>(name: string) => {
  const Context = createContext<T | undefined>(undefined)

  Context.displayName = name

  return Context
}
