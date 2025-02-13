export interface Profile {
  name: string
  description: string
  version?: string
  scripts: Script[]
}

export interface Script {
  name: string
  description?: string
  version: string
  config?: { [key: string]: any }
}

export interface Session {
  id: string
  ref: string
  created: number
  stopped: number
  files: string[]
  status: string
  results: any[]
  profile?: Profile
}
