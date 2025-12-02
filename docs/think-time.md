# Think Time

Think time simulates the pauses real users take between actions. Instead of hammering your API as fast as possible, you can add realistic delays between requests.

## Why Use Think Time?

Real users don't click instantly. They:
- Read content (2-10 seconds)
- Fill out forms (5-30 seconds)
- Think about decisions (1-5 seconds)
- Navigate between pages (1-3 seconds)

Without think time, Bombardino sends requests as fast as possible. This tests **maximum throughput**, but doesn't reflect real-world usage.

With think time, you can:
- Simulate realistic user behavior
- Test how your API handles sustained load
- Find memory leaks that only appear over time
- Measure performance under typical conditions

## When to Use Think Time

| Scenario | Think Time? |
|----------|-------------|
| Maximum throughput testing | No |
| Realistic load simulation | Yes |
| Finding rate limit issues | No |
| Simulating user sessions | Yes |
| Performance benchmarking | No |
| Capacity planning | Yes |

## Configuration Options

### Fixed Think Time

Add a consistent delay after each request:

```json
{
  "global": {
    "think_time": "500ms"
  }
}
```

After each request, Bombardino waits exactly 500ms before the next one.

### Random Range

Real users don't have consistent timing. Use a random range:

```json
{
  "global": {
    "think_time_min": "1s",
    "think_time_max": "3s"
  }
}
```

After each request, Bombardino waits a random time between 1 and 3 seconds.

### Per-Test Override

Different actions have different think times. Override at the test level:

```json
{
  "global": {
    "think_time": "2s"
  },
  "tests": [
    {
      "name": "Browse Products",
      "path": "/products",
      "think_time": "3s"
    },
    {
      "name": "Add to Cart",
      "path": "/cart",
      "think_time": "500ms"
    }
  ]
}
```

## Duration Formats

Think time uses Go duration format:

| Value | Meaning |
|-------|---------|
| `100ms` | 100 milliseconds |
| `1s` | 1 second |
| `1.5s` | 1.5 seconds |
| `2m` | 2 minutes |
| `1m30s` | 1 minute and 30 seconds |

## Example: E-Commerce User Flow

Simulate a user browsing and buying:

```json
{
  "name": "E-Commerce User Session",
  "global": {
    "base_url": "http://localhost:8080",
    "iterations": 10,
    "think_time_min": "1s",
    "think_time_max": "2s"
  },
  "tests": [
    {
      "name": "View Homepage",
      "method": "GET",
      "path": "/",
      "think_time_min": "2s",
      "think_time_max": "5s"
    },
    {
      "name": "Browse Category",
      "method": "GET",
      "path": "/products?category=electronics",
      "think_time_min": "3s",
      "think_time_max": "8s"
    },
    {
      "name": "View Product",
      "method": "GET",
      "path": "/products/123",
      "think_time_min": "5s",
      "think_time_max": "15s"
    },
    {
      "name": "Add to Cart",
      "method": "POST",
      "path": "/cart",
      "body": {"product_id": 123, "quantity": 1},
      "think_time": "500ms"
    },
    {
      "name": "Checkout",
      "method": "POST",
      "path": "/checkout",
      "think_time": "100ms"
    }
  ]
}
```

**Realistic timing:**
- Homepage: 2-5s (user reads, looks around)
- Category: 3-8s (user scans products)
- Product: 5-15s (user reads details, reviews)
- Add to Cart: 500ms (quick action)
- Checkout: 100ms (final click)

## Think Time vs Delay

Bombardino has two timing options:

| Option | Purpose |
|--------|---------|
| `delay` | Wait between ALL requests (rate limiting) |
| `think_time` | Wait after EACH test (user simulation) |

**Delay example:**
```json
{
  "global": {
    "delay": "100ms"
  }
}
```
Adds 100ms between every request, regardless of which test.

**Think time example:**
```json
{
  "global": {
    "think_time": "1s"
  }
}
```
Adds 1s after each test iteration, simulating user pause.

**Use both together:**
```json
{
  "global": {
    "delay": "50ms",
    "think_time": "1s"
  }
}
```
- Small delay between requests (rate limiting)
- Longer think time after each test (user simulation)

## Impact on Test Duration

Think time significantly affects total test duration.

**Without think time (100 iterations, 5 tests):**
```
100 iterations × 5 tests × 100ms response = ~50 seconds
```

**With 2s think time (100 iterations, 5 tests):**
```
100 iterations × 5 tests × (100ms + 2000ms) = ~1050 seconds (~17 minutes)
```

Plan accordingly!

## Tips

1. **Start without think time**: First verify your tests work correctly
2. **Add realistic times**: Base think time on real user behavior data
3. **Use ranges**: `think_time_min/max` is more realistic than fixed values
4. **Consider the flow**: Fast actions (clicks) need less time than slow ones (reading)
5. **Watch duration tests**: With `duration: "5m"`, think time still applies

## Next Steps

- [Data-Driven Testing](data-driven-testing.md) - Run tests with different data
- [Configuration Reference](configuration-reference.md) - All timing options
- [Tutorial: CRUD API](tutorial-crud-api.md) - Complete example with think time
