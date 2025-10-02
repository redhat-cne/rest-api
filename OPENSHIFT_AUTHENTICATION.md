# OpenShift Authentication Solution

This document describes the unified authentication solution for the Cloud Native Events REST API using OpenShift's built-in components. This approach works seamlessly for both single node and multi-node OpenShift clusters.

## Overview

The authentication solution leverages OpenShift's native components to provide enterprise-grade security with minimal complexity:

- **mTLS**: OpenShift Service CA for automatic certificate management
- **OAuth2**: OpenShift's built-in OAuth server with ServiceAccounts

## Why This Approach?

### Benefits for All Cluster Sizes:

| Aspect | Single Node | Multi-Node | Improvement |
|--------|-------------|------------|-------------|
| **Complexity** | ✅ Low | ✅ Low | Same simple configuration |
| **Resource Usage** | ✅ Minimal | ✅ Minimal | No additional operators |
| **High Availability** | ❌ Single point | ✅ **HA Built-in** | OAuth server runs in HA mode |
| **Performance** | ✅ Good | ✅ **Excellent** | Better throughput in multi-node |
| **Maintenance** | ✅ Automatic | ✅ Automatic | Same automation, better resilience |
| **Cost** | ✅ Free | ✅ Free | No additional licensing |

### Comparison with Alternatives:

| Approach | Single Node | Multi-Node | Complexity | Resource Usage |
|----------|-------------|------------|------------|----------------|
| **Service CA + OAuth** | ✅ **Perfect** | ✅ **Perfect** | ✅ Low | ✅ Minimal |
| cert-manager + Auth Operator | ⚠️ Overkill | ⚠️ Overkill | ❌ High | ❌ High |
| Service Mesh | ❌ Overkill | ⚠️ Overkill | ❌ Very High | ❌ Very High |
| Manual certificates | ⚠️ Maintenance burden | ❌ Poor | ❌ Very High | ⚠️ Low |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           OpenShift Cluster (Any Size)                         │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  ┌─────────────────┐              ┌─────────────────────────────────────────┐   │
│  │   Service CA    │              │     OpenShift OAuth Server (HA)        │   │
│  │                 │              │                                         │   │
│  │ • Auto certs    │              │ • High Availability                    │   │
│  │ • Auto rotation │              │ • Load balanced                        │   │
│  │ • No operators  │              │ • Multiple replicas                    │   │
│  │ • Cluster-wide  │              │ • Distributed across nodes             │   │
│  └─────────────────┘              └─────────────────────────────────────────┘   │
│           │                                       │                             │
│           ▼                                       ▼                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │              cloud-event-proxy API (DaemonSet)                         │   │
│  │                                                                         │   │
│  │  ┌─────────────┐              ┌─────────────────────┐                   │   │
│  │  │    mTLS     │              │      OAuth2         │   │
│  │  │             │              │                     │   │
│  │  │ • Client    │              │ • JWT validation    │   │
│  │  │   certs     │              │ • Scope checking    │   │
│  │  │ • Server    │              │ • Audience check    │   │
│  │  │   certs     │              │ • ServiceAccount    │   │
│  │  │ • Same on   │              │ • Same across       │   │
│  │  │   all nodes │              │   all nodes         │   │
│  │  └─────────────┘              └─────────────────────┘   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │
│  │   Node 1    │  │   Node 2    │  │   Node 3    │  │   Node N    │           │
│  │             │  │             │  │             │  │             │           │
│  │ • Same      │  │ • Same      │  │ • Same      │  │ • Same      │           │
│  │   config    │  │   config    │  │   config    │  │   config    │           │
│  │ • Same      │  │ • Same      │  │ • Same      │  │ • Same      │           │
│  │   certs     │  │   certs     │  │   certs     │  │   certs     │           │
│  │ • Same      │  │ • Same      │  │ • Same      │  │ • Same      │           │
│  │   auth      │  │   auth      │  │   auth      │  │   auth      │           │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘           │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Configuration

### Unified Configuration

The same configuration works for both single node and multi-node clusters:

```json
{
  "enableMTLS": true,
  "useServiceCA": true,
  "caCertPath": "/etc/cloud-event-proxy/ca-bundle/service-ca.crt",
  "serverCertPath": "/etc/cloud-event-proxy/server-certs/tls.crt",
  "serverKeyPath": "/etc/cloud-event-proxy/server-certs/tls.key",
  "enableOAuth": true,
  "useOpenShiftOAuth": true,
  "oauthIssuer": "https://oauth-openshift.apps.your-cluster.com",
  "oauthJWKSURL": "https://oauth-openshift.apps.your-cluster.com/.well-known/jwks.json",
  "requiredScopes": ["user:info"],
  "requiredAudience": "openshift",
  "serviceAccountName": "cloud-event-proxy-sa",
  "serviceAccountToken": "/var/run/secrets/kubernetes.io/serviceaccount/token"
}
```

### Dynamic Cluster Configuration

For environments with multiple clusters or dynamic cluster names, use the `CLUSTER_NAME` environment variable:

#### Environment Variable Configuration
```bash
# Default cluster name (used by ptp-operator)
export CLUSTER_NAME="openshift.local"

# Custom cluster name
export CLUSTER_NAME="cnfdg4.sno.ptp.eng.rdu2.dc.redhat.com"

# OAuth URLs are automatically generated as:
# https://oauth-openshift.apps.${CLUSTER_NAME}
```

#### Template Configuration
```json
{
  "enableMTLS": true,
  "useServiceCA": true,
  "caCertPath": "/etc/cloud-event-proxy/ca-bundle/service-ca.crt",
  "serverCertPath": "/etc/cloud-event-proxy/server-certs/tls.crt",
  "serverKeyPath": "/etc/cloud-event-proxy/server-certs/tls.key",
  "enableOAuth": true,
  "useOpenShiftOAuth": true,
  "oauthIssuer": "https://oauth-openshift.apps.{{.ClusterName}}",
  "oauthJWKSURL": "https://oauth-openshift.apps.{{.ClusterName}}/oauth/jwks",
  "requiredScopes": ["user:info"],
  "requiredAudience": "openshift",
  "serviceAccountName": "cloud-event-proxy-sa",
  "serviceAccountToken": "/var/run/secrets/kubernetes.io/serviceaccount/token"
}
```

This ensures OAuth issuer URLs match your actual OpenShift cluster configuration and prevents authentication bypass due to issuer mismatches.

### Key Configuration Fields

#### mTLS Configuration:
- `useServiceCA: true` - Use OpenShift Service CA (recommended for all cluster sizes)
- `caCertPath` - Path to Service CA certificate
- `serverCertPath` - Path to server certificate (auto-generated by Service CA)
- `serverKeyPath` - Path to server private key (auto-generated by Service CA)

#### OAuth Configuration:
- `useOpenShiftOAuth: true` - Use OpenShift's built-in OAuth server (recommended for all cluster sizes)
- `oauthIssuer` - OpenShift OAuth server URL
- `oauthJWKSURL` - JWKS endpoint for JWT validation
- `requiredScopes` - Required OAuth scopes
- `requiredAudience` - Required OAuth audience

## Deployment

### 1. Service with Service CA Annotation

```yaml
apiVersion: v1
kind: Service
metadata:
  name: ptp-event-publisher-service
  namespace: openshift-ptp
  annotations:
    service.beta.openshift.io/serving-cert-secret-name: cloud-event-proxy-tls
spec:
  selector:
    app: linuxptp-daemon
  ports:
  - port: 9043
    targetPort: 9043
  type: ClusterIP
```

### 2. ServiceAccount and RBAC

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cloud-event-proxy-sa
  namespace: openshift-ptp
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: cloud-event-proxy-role
  namespace: openshift-ptp
rules:
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: cloud-event-proxy-binding
  namespace: openshift-ptp
subjects:
- kind: ServiceAccount
  name: cloud-event-proxy-sa
  namespace: openshift-ptp
roleRef:
  kind: Role
  name: cloud-event-proxy-role
  apiGroup: rbac.authorization.k8s.io
```

### 3. ConfigMap with Auth Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cloud-event-proxy-auth-config
  namespace: openshift-ptp
data:
  auth-config.json: |
    {
      "enableMTLS": true,
      "useServiceCA": true,
      "caCertPath": "/etc/cloud-event-proxy/ca-bundle/service-ca.crt",
      "serverCertPath": "/etc/cloud-event-proxy/server-certs/tls.crt",
      "serverKeyPath": "/etc/cloud-event-proxy/server-certs/tls.key",
      "enableOAuth": true,
      "useOpenShiftOAuth": true,
      "oauthIssuer": "https://oauth-openshift.apps.your-cluster.com",
      "oauthJWKSURL": "https://oauth-openshift.apps.your-cluster.com/.well-known/jwks.json",
      "requiredScopes": ["user:info"],
      "requiredAudience": "openshift",
      "serviceAccountName": "cloud-event-proxy-sa",
      "serviceAccountToken": "/var/run/secrets/kubernetes.io/serviceaccount/token"
    }
```

### 4. DaemonSet with Volume Mounts

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: linuxptp-daemon
  namespace: openshift-ptp
spec:
  selector:
    matchLabels:
      app: linuxptp-daemon
  template:
    metadata:
      labels:
        app: linuxptp-daemon
    spec:
      serviceAccountName: cloud-event-proxy-sa
      containers:
      - name: cloud-event-proxy
        image: quay.io/redhat-cne/cloud-event-proxy:latest
        args:
          - "--auth-config=/etc/cloud-event-proxy/auth/auth-config.json"
        volumeMounts:
        - name: server-certs
          mountPath: /etc/cloud-event-proxy/server-certs
          readOnly: true
        - name: ca-bundle
          mountPath: /etc/cloud-event-proxy/ca-bundle
          readOnly: true
        - name: auth-config
          mountPath: /etc/cloud-event-proxy/auth
          readOnly: true
      volumes:
      - name: server-certs
        secret:
          secretName: cloud-event-proxy-tls
      - name: ca-bundle
        secret:
          secretName: cloud-event-proxy-tls
      - name: auth-config
        configMap:
          name: cloud-event-proxy-auth-config
```

## Multi-Node Benefits

### Performance Improvements:

1. **Parallel Processing**: Authentication requests processed in parallel across nodes
2. **Load Distribution**: No single node bottleneck
3. **Faster Response**: Multiple OAuth server instances
4. **Better Throughput**: Higher concurrent request handling
5. **Automatic Failover**: If one node fails, others continue serving

### High Availability:

- **OAuth Server HA**: Runs in HA mode by default
- **Service CA Resilience**: Certificate authority is cluster-wide
- **No Single Points of Failure**: Distributed across multiple nodes

## Client Configuration

### For Clients in the Same Cluster:

```json
{
  "enableMTLS": true,
  "useServiceCA": true,
  "caCertPath": "/etc/cloud-event-consumer/ca-bundle/service-ca.crt",
  "clientCertPath": "/etc/cloud-event-consumer/client-certs/tls.crt",
  "clientKeyPath": "/etc/cloud-event-consumer/client-certs/tls.key",
  "enableOAuth": true,
  "useOpenShiftOAuth": true,
  "oauthIssuer": "https://oauth-openshift.apps.your-cluster.com",
  "oauthJWKSURL": "https://oauth-openshift.apps.your-cluster.com/.well-known/jwks.json",
  "requiredScopes": ["user:info"],
  "requiredAudience": "openshift",
  "serviceAccountName": "consumer-sa",
  "serviceAccountToken": "/var/run/secrets/kubernetes.io/serviceaccount/token"
}
```

## Troubleshooting

### Common Issues:

1. **Certificate Not Found**: Ensure Service CA annotation is correct
2. **OAuth Validation Fails**: Check OAuth server URL and JWKS endpoint
3. **Permission Denied**: Verify ServiceAccount has proper RBAC permissions

### Debug Commands:

```bash
# Check if Service CA secret exists
oc get secret cloud-event-proxy-tls -n openshift-ptp

# Check ServiceAccount token
oc get secret -n openshift-ptp -o name | grep cloud-event-proxy-sa

# Verify OAuth server accessibility
curl -k https://oauth-openshift.apps.your-cluster.com/.well-known/jwks.json

# Check OAuth server HA status
oc get deployment oauth-openshift -n openshift-authentication

# Monitor authentication performance
oc top pods -n openshift-authentication
oc top pods -n openshift-ptp
```

## Migration

### From Other Approaches:

#### From Manual Certificates:
1. Set `useServiceCA: true`
2. Remove manual certificate generation scripts
3. Update certificate paths to use Service CA secrets

#### From Service Mesh:
1. Remove Service Mesh configuration
2. Use this Service CA + OAuth approach
3. Update client configurations accordingly

### Scaling Up:

1. **No Configuration Changes**: Same configuration works for multi-node
2. **Automatic Scaling**: DaemonSet automatically deploys to new nodes
3. **HA Benefits**: Automatically get high availability benefits
4. **Performance Improvement**: Better performance without changes

## Best Practices

1. **Use DaemonSet**: Ensures consistent deployment across nodes
2. **Monitor OAuth Server**: Keep an eye on OAuth server performance
3. **Resource Planning**: Plan for increased resource usage in multi-node
4. **Network Policies**: Consider network policies for inter-node communication
5. **Regular Updates**: Keep OpenShift cluster updated for security patches

## Conclusion

The Service CA + OpenShift OAuth approach provides:

- ✅ **Unified Solution**: Same configuration for single and multi-node clusters
- ✅ **Better Performance**: Load distribution and parallel processing
- ✅ **High Availability**: Built-in HA for OAuth server
- ✅ **Simplified Management**: Same configuration across all nodes
- ✅ **Automatic Scaling**: Scales with cluster size
- ✅ **Enterprise Security**: Consistent security across cluster
- ✅ **Cost Effective**: No additional licensing or resource costs

This approach scales from single node to large multi-node clusters without any configuration changes, making it the ideal solution for OpenShift deployments of any size.
