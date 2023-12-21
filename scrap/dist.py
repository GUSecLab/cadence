#constants
deltat=10.

a=1.
b=6.
c=1.
d=6.

e=2.
f=4.
g=2.
h=4.



# derived

alpha=a-e
beta=c-g
delta1=b-f
delta2=d-h

num = (alpha * delta1) + (beta * delta2)
denom = (alpha * alpha) + (beta * beta)

if denom == 0:
    print( 'distance remains constant during interval' )
    exit(0)
    
tmin = min(deltat,max(0,-1. * num / denom))


print(f'time of encounter: {tmin}')

x1 = (a * tmin) + b
y1 = (c * tmin) + d
x2 = (e * tmin) + f
y2 = (g * tmin) + h

dist = (((x1-x2)**2.) + ((y1-y2)**2.)) ** .5
    
print( f'location of node1:      ({x1},{y1})')
print( f'location of node2:      ({x2},{y2})')
print( f'distance between nodes: {dist}')
