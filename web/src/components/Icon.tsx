import type { SVGProps } from 'react'

export type IconName =
  | 'add'
  | 'alert'
  | 'architecture'
  | 'arrow-right'
  | 'check'
  | 'chevron-down'
  | 'chevron-right'
  | 'close'
  | 'code'
  | 'external'
  | 'file'
  | 'filter'
  | 'flow'
  | 'folder'
  | 'impact'
  | 'layers'
  | 'lock'
  | 'menu'
  | 'power'
  | 'refresh'
  | 'repository'
  | 'route'
  | 'search'
  | 'shield'
  | 'spark'
  | 'test'
  | 'trash'

export function Icon({ name, ...props }: SVGProps<SVGSVGElement> & { name: IconName }) {
  const content = iconContent(name)
  return (
    <svg aria-hidden="true" fill="none" height="20" viewBox="0 0 24 24" width="20" {...props}>
      {content}
    </svg>
  )
}

function iconContent(name: IconName) {
  const shared = {
    stroke: 'currentColor',
    strokeLinecap: 'round' as const,
    strokeLinejoin: 'round' as const,
    strokeWidth: 1.8,
  }
  switch (name) {
    case 'add':
      return <path {...shared} d="M12 5v14M5 12h14" />
    case 'alert':
      return (
        <>
          <path {...shared} d="M12 9v4m0 4h.01" />
          <path
            {...shared}
            d="M10.3 4.6 2.8 17.4A2 2 0 0 0 4.5 20h15a2 2 0 0 0 1.7-2.6L13.7 4.6a2 2 0 0 0-3.4 0Z"
          />
        </>
      )
    case 'architecture':
      return (
        <path {...shared} d="M4 5h6v5H4V5Zm10 9h6v5h-6v-5Zm0-9h6v5h-6V5ZM7 10v4h10v-4M7 14v3h7" />
      )
    case 'arrow-right':
      return <path {...shared} d="M5 12h14m-6-6 6 6-6 6" />
    case 'check':
      return <path {...shared} d="m5 12 4 4L19 6" />
    case 'chevron-down':
      return <path {...shared} d="m7 9 5 5 5-5" />
    case 'chevron-right':
      return <path {...shared} d="m9 6 6 6-6 6" />
    case 'close':
      return <path {...shared} d="M6 6l12 12M18 6 6 18" />
    case 'code':
      return <path {...shared} d="m8 9-3 3 3 3m8-6 3 3-3 3m-2-9-4 12" />
    case 'external':
      return (
        <path
          {...shared}
          d="M14 5h5v5m0-5-8 8M19 14v4a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h4"
        />
      )
    case 'file':
      return <path {...shared} d="M7 3h7l4 4v14H7V3Zm7 0v5h4M10 13h5m-5 4h5" />
    case 'filter':
      return <path {...shared} d="M4 6h16M7 12h10m-7 6h4" />
    case 'flow':
      return (
        <path
          {...shared}
          d="M5 5h5v5H5V5Zm9 9h5v5h-5v-5ZM10 7.5h3a3 3 0 0 1 3 3V14m-8 0v5m-3-2h6"
        />
      )
    case 'folder':
      return <path {...shared} d="M3 6h7l2 2h9v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6Z" />
    case 'impact':
      return (
        <>
          <circle {...shared} cx="12" cy="12" r="3" />
          <circle {...shared} cx="12" cy="12" r="8" />
          <path {...shared} d="M12 1v3m0 16v3M1 12h3m16 0h3" />
        </>
      )
    case 'layers':
      return <path {...shared} d="m12 3 9 5-9 5-9-5 9-5Zm-7 9 7 4 7-4M5 16l7 4 7-4" />
    case 'lock':
      return <path {...shared} d="M7 10V7a5 5 0 0 1 10 0v3m-11 0h12v10H6V10Zm6 4v2" />
    case 'menu':
      return <path {...shared} d="M5 7h14M5 12h14M5 17h14" />
    case 'power':
      return <path {...shared} d="M12 3v9m5.7-6.7a9 9 0 1 1-11.4 0" />
    case 'refresh':
      return (
        <path {...shared} d="M20 7v5h-5M4 17v-5h5m9.4-3A7 7 0 0 0 6.1 7M5.6 15A7 7 0 0 0 17.9 17" />
      )
    case 'repository':
      return <path {...shared} d="M5 3h11a3 3 0 0 1 3 3v15H7a2 2 0 0 1-2-2V3Zm0 14h14M9 7h6" />
    case 'route':
      return (
        <>
          <circle {...shared} cx="6" cy="18" r="2" />
          <circle {...shared} cx="18" cy="6" r="2" />
          <path {...shared} d="M8 18h3a3 3 0 0 0 3-3V9a3 3 0 0 1 3-3" />
        </>
      )
    case 'search':
      return <path {...shared} d="m20 20-4.4-4.4m2.4-5.1a7.5 7.5 0 1 1-15 0 7.5 7.5 0 0 1 15 0Z" />
    case 'shield':
      return (
        <path
          {...shared}
          d="M12 3 5 6v5c0 4.6 2.8 7.8 7 10 4.2-2.2 7-5.4 7-10V6l-7-3Zm-3 9 2 2 4-5"
        />
      )
    case 'spark':
      return (
        <path
          {...shared}
          d="m12 3 1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3Zm6 12 .8 2.2L21 18l-2.2.8L18 21l-.8-2.2L15 18l2.2-.8L18 15Z"
        />
      )
    case 'test':
      return (
        <path
          {...shared}
          d="M9 3h6m-5 0v5l-5 9a2 2 0 0 0 1.7 3h10.6a2 2 0 0 0 1.7-3l-5-9V3M8 14h8"
        />
      )
    case 'trash':
      return <path {...shared} d="M4 7h16m-10 4v5m4-5v5M7 7l1 13h8l1-13M9 7V4h6v3" />
  }
}
