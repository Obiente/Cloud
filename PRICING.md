# Obiente Cloud Pricing

## Overview

Obiente Cloud offers competitive, pay-as-you-go pricing designed to match traditional VPS providers while providing the flexibility of cloud-native resource allocation. Our pricing is competitive with major VPS providers like DigitalOcean, Linode, and Vultr.

**Perfect for game servers and VPSs:** Traditional hosting providers charge you for full-time resources even when your server is idle or offline. With our pay-as-you-go model, you only pay for actual runtime and resource usage, saving money when your game server is offline or your VPS has low utilization.

**Note:** As we grow and achieve better economies of scale, we plan to reduce pricing for storage and other resources. We're committed to passing cost savings along to our customers.

## Pricing Model

All resources are billed based on actual usage with no upfront costs or minimum commitments. Perfect for applications that need VPS-like pricing with cloud flexibility.

**Why pay-as-you-go matters:**
- **Game servers:** Low costs when idle or offline. Traditional hosting charges $20-40/month even when your server is empty.
- **VPS instances:** Pay for actual CPU and memory usage, not idle time. Most VPS providers charge full price regardless of utilization.
- **Development environments:** Stop paying for resources that sit idle overnight or on weekends.
- **Variable workloads:** Scale costs automatically with demand - no over-provisioning required.

---

## Resource Pricing

### 💾 Memory (RAM)

**$0.00411 per GB-hour** ($3.00 per GB-month for 24/7 usage)

- Billed per byte-second of memory usage
- Only charged when your deployments are running
- Based on actual memory consumption, not allocated memory

**Example:**

- 1 GB deployment running for 1 hour = **$0.00411**
- 1 GB deployment running 24/7 for a month (730 hours) = **$3.00**
- 512 MB deployment running 24/7 for a month = **$1.50**

### ⚡ vCPU

**$0.00274 per vCPU-hour** ($2.00 per vCPU-month for 24/7 usage)

- Billed per core-second of vCPU usage
- Calculated based on actual CPU utilization percentage
- Scales automatically with your workload

**Example:**

- 1 vCPU running at 100% for 1 hour = **$0.00274**
- 1 vCPU running at 50% for 1 hour = **$0.00137**
- 2 vCPU cores running at 100% for 1 hour = **$0.00548**

### 🌐 Bandwidth

**$0.01 per GB**

- Billed per byte transferred (inbound + outbound)
- No charges for internal network traffic
- Very affordable, similar to VPS providers

**Example:**

- 10 GB transferred in a month = **$0.10**
- 100 GB transferred in a month = **$1.00**
- 1 TB transferred in a month = **$10.00**

### 💿 Storage

**$0.20 per GB-month**

- Billed monthly for persistent storage
- Includes Docker images, volumes, and container filesystems
- Prorated for partial months
- _Note: Higher pricing reflects our limited storage capacity - we focus on compute resources, not storage. Storage pricing will decrease as we scale our infrastructure._

**Example:**

- 10 GB stored for 1 month = **$2.00**
- 25 GB stored for 1 month = **$5.00**
- 100 GB stored for 1 month = **$20.00**

---

## Example Scenarios

### Small Application

- **512 MB RAM** running 24/7
- **0.25 vCPU cores** average utilization
- **10 GB bandwidth** per month
- **5 GB storage**

**Monthly Cost: ~$5.35**

- Memory: $1.50
- vCPU: $1.37
- Bandwidth: $0.10
- Storage: $1.00

### Medium Application

- **2 GB RAM** running 24/7
- **1 vCPU core** average utilization
- **50 GB bandwidth** per month
- **25 GB storage**

**Monthly Cost: ~$14.00**

- Memory: $6.00
- vCPU: $2.00
- Bandwidth: $0.50
- Storage: $5.00

### Large Application

- **8 GB RAM** running 24/7
- **2 vCPU cores** average utilization
- **200 GB bandwidth** per month
- **100 GB storage**

**Monthly Cost: ~$44.00**

- Memory: $24.00
- vCPU: $4.00
- Bandwidth: $2.00
- Storage: $20.00

### Game Server Example

- **4 GB RAM** running 12 hours/day (50% uptime)
- **2 vCPU cores** average utilization
- **100 GB bandwidth** per month
- **20 GB storage**

**Monthly Cost: ~$15.00**

- Memory: $6.00 (12h/day = 50% of 24/7)
- vCPU: $2.00 (12h/day = 50% of 24/7)
- Bandwidth: $1.00
- Storage: $4.00

**Compare to traditional hosting:** A typical game server hosting plan charges $20-30/month for this configuration, even when your server is offline. With pay-as-you-go, you save 50% by paying low costs when idle or offline.

### VPS Example

- **2 GB RAM** running 24/7
- **1 vCPU core** average utilization
- **50 GB bandwidth** per month
- **10 GB storage**

**Monthly Cost: ~$8.00**

- Memory: $6.00
- vCPU: $2.00
- Bandwidth: $0.50
- Storage: $2.00

**Compare to traditional VPS:** Similar VPS plans cost $10-12/month regardless of usage. If your VPS runs part-time or has low utilization, pay-as-you-go pricing saves you money.

---

## Comparison with VPS Providers

| Resource                   | Obiente Cloud    | DigitalOcean    | Linode          | Vultr           |
| -------------------------- | ---------------- | --------------- | --------------- | --------------- |
| **1GB RAM + 1 vCPU** (24/7) | **$5.00/month**  | $4-6/month      | $5/month        | $4-6/month      |
| **2GB RAM + 2 vCPU** (24/7) | **$10.00/month** | $12/month       | $10/month       | $12/month       |
| **Bandwidth**              | $0.01/GB         | Included (1TB)  | Included (1TB)  | Included (1TB)  |
| **Storage**                | $0.20/GB-month   | Included (25GB) | Included (25GB) | Included (10GB) |

_Note: VPS providers typically bundle bandwidth and storage, while we charge separately for more flexible pricing. Our pay-as-you-go model means you only pay for what you use. Storage pricing is higher to reflect our limited capacity - we focus on compute resources._

---

## How Billing Works

1. **Real-time Usage Tracking**: All resources are tracked in real-time using metrics collected every 5 seconds
2. **Hourly Aggregation**: Usage is aggregated hourly for efficient storage and calculation
3. **Monthly Billing**: Costs are calculated monthly and displayed in your dashboard
4. **Cost Breakdown**: View detailed cost breakdowns by resource type (vCPU, Memory, Bandwidth, Storage)
5. **Estimated Costs**: See projected monthly costs based on current usage patterns

### Cost Calculation Details

- **Memory & vCPU**: Billed based on actual usage (byte-seconds and core-seconds)
- **Bandwidth**: Billed per byte transferred (cumulative)
- **Storage**: Billed monthly based on snapshot at end of month (prorated for partial months)

---

## Cost Optimization Tips

1. **Right-size your deployments**: Match memory and vCPU allocations to actual needs
2. **Monitor usage**: Use the dashboard to identify resource-heavy deployments
3. **Optimize storage**: Clean up unused Docker images and volumes regularly
4. **Cache wisely**: Reduce bandwidth costs by implementing proper caching strategies
5. **Scale smartly**: Use auto-scaling features to match demand without over-provisioning

---

## FAQ

**Q: Is there a minimum charge or setup fee?**  
A: No. We only charge for actual resource usage with no minimums or setup fees.

**Q: How does this compare to traditional VPS pricing?**  
A: Our pricing is competitive with VPS providers like DigitalOcean and Linode. For example, 1GB RAM + 1 vCPU running 24/7 costs $5/month, similar to entry-level VPS plans. The key advantage is you only pay for what you use - if your VPS runs part-time or has low utilization, you pay less than traditional fixed-price plans.

**Q: How accurate are the cost estimates?**  
A: Estimates are based on your actual usage patterns. For the current month, estimates project usage based on elapsed time. Historical months show actual costs.

**Q: Can I set spending limits?**  
A: Spending limits and budget alerts are available in your organization settings.

**Q: How do I view my billing history?**  
A: View detailed billing history and cost breakdowns in your dashboard under Billing.

**Q: Are there any free plans?**  
A: We offer free plans for certain customers (students, open-source projects, non-profits, and early-stage startups). Please reach out to our team for more details. Free plans are not available through the dashboard - you must contact us to discuss eligibility.

**Q: What if my deployment only runs part-time?**  
A: You only pay for actual runtime. If your deployment runs 12 hours/day instead of 24/7, you'll pay approximately half the monthly cost. This is a key advantage over traditional VPS where you pay for the full month regardless of usage. Perfect for game servers that are only active during peak hours or development environments used only during work hours.

**Q: How much can I save with pay-as-you-go vs traditional hosting?**  
A: Savings depend on your usage patterns. For game servers running 12 hours/day, you can save 50% compared to fixed-price hosting. For VPSs with low CPU utilization (under 50%), savings can be 30-40%. Development environments used only during work hours can save 60-70% compared to 24/7 hosting.

**Q: Will pricing change over time?**  
A: Yes, we plan to reduce pricing for storage and other resources as we grow and achieve better economies of scale. We're committed to passing cost savings along to our customers.

---

## Free Plans

We offer free plans for qualifying customers, including:

- **Students** - Educational projects and coursework
- **Open-source projects** - Non-commercial open-source initiatives
- **Non-profits** - Registered non-profit organizations
- **Early-stage startups** - Pre-revenue startups and MVPs

Free plans are not available through the dashboard - please contact us to discuss your eligibility and requirements. We'll work with you to determine the best plan for your needs.

---

## Contact

For pricing questions, custom enterprise pricing, or to inquire about free plans, please contact our sales team or reach out through your dashboard.

---

_Last updated: November 2025_  
_Pricing subject to change. All prices in USD. We plan to reduce pricing as we scale - cost savings will be passed along to customers._
