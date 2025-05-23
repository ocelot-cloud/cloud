<template>
  <v-text-field
      :id="id"
      :type="inputType"
      v-model="localValue"
      @input="updateValue"
      :error="submitted && hasError"
      :error-messages="submitted && hasError ? [errorMessageText] : []"
      :placeholder="label"
      class="mb-1"
      required
  />
</template>

<script lang="ts">
import {defineComponent, computed, watch, ref} from 'vue'

type ValidationType = 'default' | 'password' | 'appSearch' | 'host' | 'number' | 'email_or_empty'

export const defaultAllowedSymbols = '[0-9a-z]';
export const passwordAllowedSymbols = '[a-zA-Z0-9._-]';
export const defaultMinLength = 3;
export const defaultMaxLength = 20;
export const minLengthPassword = 8;
export const maxLengthPassword = 30;
export const searchBarMinLength = 0;

export function generateInvalidInputMessage(allowedSymbols: string, minLength: number, maxLength: number): string {
  return `Invalid input, allowed symbols are ${allowedSymbols} and the length must range from ${minLength} to ${maxLength}.`;
}

export function getDefaultValidationRegex(): RegExp {
  return createRegex(defaultAllowedSymbols, defaultMinLength, defaultMaxLength)
}

export function createRegex(allowedSymbols: string, minLength: number, maxLength: number): RegExp {
  return new RegExp(`^${allowedSymbols}{${minLength},${maxLength}}$`)
}

const validationConfig = {
  default: {
    type: 'text',
    pattern: getDefaultValidationRegex(),
    errorMessage: generateInvalidInputMessage(defaultAllowedSymbols, defaultMinLength, defaultMaxLength),
  },
  password: {
    type: 'password',
    pattern: createRegex(passwordAllowedSymbols, minLengthPassword, maxLengthPassword),
    errorMessage: generateInvalidInputMessage(passwordAllowedSymbols, minLengthPassword, maxLengthPassword),
  },
  appSearch: {
    type: 'text',
    pattern: createRegex(defaultAllowedSymbols, searchBarMinLength, defaultMaxLength),
    errorMessage: generateInvalidInputMessage(defaultAllowedSymbols, searchBarMinLength, defaultMaxLength),
  },
  host: {
    type: 'text',
    pattern: createRegex('[a-zA-Z0-9:._-]', 1, 64),
    errorMessage: generateInvalidInputMessage('[a-zA-Z0-9:._-]', 1, 64),
  },
  number: {
    type: 'number',
    pattern: createRegex('[0-9]', 1, 20),
    errorMessage: generateInvalidInputMessage('[0-9]', 1, 20),
  },
  email_or_empty: {
    type: 'email_or_empty',
    pattern: new RegExp('^$|^[a-zA-Z0-9._-]+@[a-zA-Z0-9._-]+\.[a-zA-Z]{2,}$'),
    errorMessage: 'empty field or email format needed'
  },
}

export default defineComponent({
  name: 'ValidatedInput',
  props: {
    modelValue: {
      type: String,
      required: true
    },
    validationType: {
      type: String as () => ValidationType,
      required: true
    },
    submitted: {
      type: Boolean,
      required: true
    },
    label: {
      type: String,
      default: ''
    },
    id: {
      type: String,
      default: ''
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    const config = validationConfig[props.validationType as ValidationType]
    const hasError = computed(() => !config.pattern.test(props.modelValue))
    const inputType = computed(() => config.type)
    const errorMessageText = computed(() => config.errorMessage)

    const localValue = ref(props.modelValue)
    watch(() => props.modelValue, v => { localValue.value = v })
    watch(localValue, (newValue) => {
      emit('update:modelValue', newValue)
    })

    const updateValue = (event: Event) => {
      emit('update:modelValue', (event.target as HTMLInputElement).value)
    }

    return {
      hasError,
      inputType,
      errorMessageText,
      updateValue,
      localValue,
    }
  }
})
</script>
