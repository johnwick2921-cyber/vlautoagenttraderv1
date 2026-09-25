import math
def wilson(k,n,z=1.96):
    if n==0: return (float('nan'),float('nan'))
    p=k/n; d=1+z*z/n; c=p+z*z/(2*n); h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))
    return ((c-h)/d,(c+h)/d)
def mean_ci(xs,z=1.96):
    n=len(xs); 
    if n<2: return (float('nan'),)*3
    m=sum(xs)/n; sd=math.sqrt(sum((x-m)**2 for x in xs)/(n-1)); se=sd/math.sqrt(n)
    return (m, m-z*se, m+z*se)
