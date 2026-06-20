# moneycli

A CLI that fetches your Revolut balance.

## The problem this repo documents

This project started simple: a CLI to check your Revolut balance from the terminal. What it became is a case study in how broken financial data access is for individuals in 2026.

### The three approaches, all broken

**1. Screen-scraping (chromedp)**
Open a browser, intercept the Bearer token from API calls, store it in the keychain. Works perfectly. Risks getting your account flagged for violating terms of service. Not a real solution.

**2. Open Banking / PSD2 (GoCardless/Nordigen)**
The "legitimate" path — uses regulated APIs with user consent. Requires going through a regulated AISP (Account Information Service Provider) like GoCardless. Those AISPs require KYB (Know Your Business) even if you just want to access your own accounts. PSD2 intended to give individuals control but the implementation requires a business registration to get started.

**3. Third-party aggregators (Plaid, Teller, Salt Edge)**
Also sit between you and the bank. All require payment (Plaid) or don't support Revolut (Teller), or still need business verification (Salt Edge).

### The gap

There is no viable, clean path for an individual to programmatically access their own bank account balance. The regulatory framework meant to enable this (PSD2) handed control to businesses, not people. The only options are scraping (ToS violation), paying a middleman, or having a business entity.

This is why Bitcoin maximalism exists — when the legacy system says "not your keys, not your money" and the regulated system says "not a business, not your API access," people start looking for alternatives.

## Usage

```bash
# Optional: mock mode for testing
MOCK=true money

# Real mode requires GoCardless API credentials
# Set GOCARDLESS_SECRET_ID and GOCARDLESS_SECRET_KEY
money
```
