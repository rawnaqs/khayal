import { describe, it, expect } from 'vitest'
import {
  STORAGE_KEYS,
  SEARCH_SUGGESTIONS,
  PROCESSING_STEPS,
  LIMITS,
  TIMEOUTS,
  GREETINGS,
  TYPE_FILTERS,
  SEARCH_MODES,
} from '../constants'

describe('constants.ts', () => {
  describe('STORAGE_KEYS', () => {
    it('should have TOKEN, HOST, and RECENT_SEARCHES keys', () => {
      expect(STORAGE_KEYS.TOKEN).toBe('khayal_token')
      expect(STORAGE_KEYS.HOST).toBe('khayal_host')
      expect(STORAGE_KEYS.RECENT_SEARCHES).toBe('khayal-recent-searches')
    })
  })

  describe('SEARCH_SUGGESTIONS', () => {
    it('should be an array of strings', () => {
      expect(Array.isArray(SEARCH_SUGGESTIONS)).toBe(true)
      expect(SEARCH_SUGGESTIONS.length).toBeGreaterThan(0)
      SEARCH_SUGGESTIONS.forEach((suggestion) => {
        expect(typeof suggestion).toBe('string')
      })
    })
  })

  describe('PROCESSING_STEPS', () => {
    it('should have steps for text, image, and article', () => {
      expect(PROCESSING_STEPS.text).toBeDefined()
      expect(PROCESSING_STEPS.image).toBeDefined()
      expect(PROCESSING_STEPS.article).toBeDefined()
    })

    it('should have saved as first step for all types', () => {
      expect(PROCESSING_STEPS.text[0]).toBe('saved')
      expect(PROCESSING_STEPS.image[0]).toBe('saved')
      expect(PROCESSING_STEPS.article[0]).toBe('saved')
    })

    it('should have writing as last step for text and article', () => {
      const textSteps = PROCESSING_STEPS.text
      const articleSteps = PROCESSING_STEPS.article
      expect(textSteps[textSteps.length - 1]).toBe('writing')
      expect(articleSteps[articleSteps.length - 1]).toBe('writing')
    })
  })

  describe('LIMITS', () => {
    it('should have SEARCH_RESULTS, QUEUE_JOBS, and RECENT_SEARCHES', () => {
      expect(LIMITS.SEARCH_RESULTS).toBe(20)
      expect(LIMITS.QUEUE_JOBS).toBe(50)
      expect(LIMITS.RECENT_SEARCHES).toBe(10)
    })

    it('should have DONE_JOBS_SHOWN, TAGS_HERO, TAGS_COMPACT', () => {
      expect(LIMITS.DONE_JOBS_SHOWN).toBe(5)
      expect(LIMITS.TAGS_HERO).toBe(3)
      expect(LIMITS.TAGS_COMPACT).toBe(2)
    })

    it('should have HERO_SCORE_THRESHOLD between 0 and 1', () => {
      expect(LIMITS.HERO_SCORE_THRESHOLD).toBeGreaterThan(0)
      expect(LIMITS.HERO_SCORE_THRESHOLD).toBeLessThanOrEqual(1)
    })
  })

  describe('TIMEOUTS', () => {
    it('should have CAPTURE_DISMISS, STATS_POLL, and SERVER_STATUS_POLL', () => {
      expect(TIMEOUTS.CAPTURE_DISMISS).toBe(3500)
      expect(TIMEOUTS.STATS_POLL).toBe(60000)
      expect(TIMEOUTS.SERVER_STATUS_POLL).toBe(30000)
    })

    it('should have all values as positive numbers', () => {
      Object.values(TIMEOUTS).forEach((timeout) => {
        expect(timeout).toBeGreaterThan(0)
      })
    })
  })

  describe('GREETINGS', () => {
    it('should have greetings for different time periods', () => {
      expect(GREETINGS.length).toBeGreaterThan(0)
      GREETINGS.forEach((greeting) => {
        expect(greeting.maxHour).toBeDefined()
        expect(greeting.text).toBeDefined()
        expect(typeof greeting.text).toBe('string')
      })
    })

    it('should cover full 24-hour range', () => {
      expect(GREETINGS[GREETINGS.length - 1].maxHour).toBe(24)
    })
  })

  describe('TYPE_FILTERS', () => {
    it('should include all, text, article, and image', () => {
      expect(TYPE_FILTERS).toContain('all')
      expect(TYPE_FILTERS).toContain('text')
      expect(TYPE_FILTERS).toContain('article')
      expect(TYPE_FILTERS).toContain('image')
    })
  })

  describe('SEARCH_MODES', () => {
    it('should include hybrid, keyword, and semantic', () => {
      expect(SEARCH_MODES).toContain('hybrid')
      expect(SEARCH_MODES).toContain('keyword')
      expect(SEARCH_MODES).toContain('semantic')
    })
  })
})
