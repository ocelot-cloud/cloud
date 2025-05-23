import { switchIsDemoDomain } from '@/components/config'

const pressedKeys = new Set<string>()

const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key) pressedKeys.add(e.key.toLowerCase())
    if (pressedKeys.has('a') && pressedKeys.has('b') && pressedKeys.has('o')) {
        switchIsDemoDomain()
    }
}

const handleKeyUp = (e: KeyboardEvent) => {
    if (e.key) pressedKeys.delete(e.key.toLowerCase())
}

export const installKeyboardListener = () => {
    window.addEventListener('keydown', handleKeyDown, { passive: false })
    window.addEventListener('keyup', handleKeyUp, { passive: false })
}
